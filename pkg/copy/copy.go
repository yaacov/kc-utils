// Package copy provides standalone vSphere disk copy via govmomi NFC export.
// It downloads VMDK disks from vCenter/ESXi over HTTPS (no VDDK required)
// and writes raw data to PVC targets (block devices or filesystem images).
package copy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sync"
)

const DefaultCopyConcurrency = 4

// CopyInput is the standalone input to Run.
type CopyInput struct {
	VCenterURL      string   `json:"vcenter_url"`
	VMName          string   `json:"vm_name"`
	VMMoref         string   `json:"vm_moref,omitempty"`
	Fingerprint     string   `json:"fingerprint"`
	SourceDisks     []string `json:"source_disks"` // VMDK paths to copy; filters NFC lease (list order → PVC index)
	Workdir         string   `json:"workdir"`
	CopyConcurrency int      `json:"copy_concurrency,omitempty"`
}

// Progress tracks per-disk copy status.
type Progress struct {
	Disks []DiskProgress `json:"disks"`
}

// DiskProgress is one disk copy result.
type DiskProgress struct {
	Index      int    `json:"index"`
	SourceFile string `json:"source_file"`
	Target     string `json:"target"`
	Status     string `json:"status"`
}

// ClampConcurrency returns a sane worker count for parallel disk copy.
func ClampConcurrency(n, disks int) int {
	if n <= 0 {
		n = DefaultCopyConcurrency
	}
	if disks < 1 {
		return 1
	}
	if n > disks {
		return disks
	}
	return n
}

// Run copies vSphere disks into empty PVC targets via NFC export.
func Run(input *CopyInput) error {
	if input.VCenterURL == "" {
		return fmt.Errorf("vcenter_url is required")
	}
	if input.VMName == "" {
		return fmt.Errorf("vm_name is required")
	}
	if input.Fingerprint == "" {
		return fmt.Errorf("fingerprint is required")
	}
	if len(input.SourceDisks) == 0 {
		return fmt.Errorf("source_disks is required")
	}

	allTargets, err := DiscoverTargets()
	if err != nil {
		return err
	}
	if err := logDiscoveredTargets(allTargets); err != nil {
		return err
	}

	targets, err := EmptyTargets()
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return fmt.Errorf("no empty PVC targets found")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	slog.Info("NFC disk copy starting",
		"source_disks", len(input.SourceDisks),
		"empty_targets", len(targets),
		"vm", input.VMName,
	)

	lease, err := ExportVM(ctx, input.VCenterURL, input.VMName)
	if err != nil {
		return fmt.Errorf("NFC export: %w", err)
	}

	selected, err := FilterDiskURLs(lease.DiskURLs, input.SourceDisks)
	if err != nil {
		_ = lease.Abort(ctx)
		return fmt.Errorf("filter NFC disks: %w", err)
	}
	if len(selected) != len(targets) {
		_ = lease.Abort(ctx)
		return fmt.Errorf("disk count mismatch: %d selected source disk(s) vs %d empty target(s)", len(selected), len(targets))
	}

	total := len(targets)
	concurrency := ClampConcurrency(input.CopyConcurrency, total)
	slog.Info("NFC disks selected",
		"selected", total,
		"lease", len(lease.DiskURLs),
		"concurrency", concurrency,
	)

	httpClient := newInsecureHTTPClient()
	progress := Progress{}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	var errOnce sync.Once

	for i, target := range targets {
		i, target := i, target
		diskURL := selected[i]
		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			if ctx.Err() != nil {
				return
			}

			slog.Info("copying disk via NFC",
				"disk", fmt.Sprintf("%d/%d", i+1, total),
				"index", target.Index,
				"source", diskURL.DiskPath,
				"target", target.Path,
				"block", target.IsBlockDev,
			)
			if err := CopyDisk(ctx, httpClient, diskURL, target, func(pct int) {
				slog.Info("disk copy progress",
					"index", target.Index,
					"source", diskURL.DiskPath,
					"target", target.Path,
					"percent", pct,
				)
			}); err != nil {
				errOnce.Do(func() {
					firstErr = err
					cancel()
				})
				return
			}

			// Reclaim per-disk buffers (compBuf, grainBuf, decompressor)
			// before the semaphore slot is released to the next disk.
			runtime.GC()

			mu.Lock()
			defer mu.Unlock()
			progress.Disks = append(progress.Disks, DiskProgress{
				Index:      target.Index,
				SourceFile: diskURL.DiskPath,
				Target:     target.Path,
				Status:     "complete",
			})
			if err := writeProgress(input, progress); err != nil {
				slog.Warn("failed to write copy progress", "error", err)
			}
			slog.Info("disk copy recorded",
				"completed", fmt.Sprintf("%d/%d", len(progress.Disks), total),
				"index", target.Index,
			)
		}()
	}

	wg.Wait()

	if firstErr != nil {
		_ = lease.Abort(ctx)
		return firstErr
	}

	if err := lease.Complete(ctx); err != nil {
		slog.Warn("NFC lease complete failed", "error", err)
	}

	slog.Info("NFC disk copy complete", "disks", total)
	return nil
}

func writeProgress(input *CopyInput, p Progress) error {
	path := input.Workdir + "/copy-progress.json"
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func logDiscoveredTargets(targets []Target) error {
	slog.Info("discovered PVC targets", "count", len(targets))
	for _, t := range targets {
		empty, err := isTargetEmpty(t)
		if err != nil {
			return err
		}
		kind := "filesystem"
		if t.IsBlockDev {
			kind = "block"
		}
		slog.Info("copy target",
			"index", t.Index,
			"path", t.Path,
			"kind", kind,
			"empty", empty,
		)
	}
	return nil
}
