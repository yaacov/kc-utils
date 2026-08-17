// Package copy provides standalone vSphere disk copy via govmomi NFC export.
// It downloads VMDK disks from vCenter/ESXi over HTTPS (no VDDK required)
// and writes raw data to PVC targets (block devices or filesystem images).
package copy

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/yaacov/kc-utils/pkg/common/types"
	v2vtls "github.com/yaacov/kc-utils/pkg/v2v/tls"
)

const DefaultCopyConcurrency = 4

// DefaultWorkdir is the default working directory for copy progress files.
const DefaultWorkdir = "/var/tmp/v2v"

// CopyInput is the standalone input to Run.
type CopyInput struct {
	Host            string   `json:"host"`
	Datacenter      string   `json:"datacenter,omitempty"`
	Insecure        bool     `json:"insecure"`
	CaCert          string   `json:"ca_cert,omitempty"`
	VMName          string   `json:"vm_name"`
	Fingerprint     string   `json:"fingerprint"`
	SourceDisks     []string `json:"source_disks,omitempty"` // VMDK paths to copy; filters NFC lease (empty = all disks)
	Workdir         string   `json:"workdir"`
	OutputPath      string   `json:"output_path,omitempty"`
	OutputDir       string   `json:"output_dir,omitempty"` // Write raw images to this dir (disk0.img, disk1.img, …); bypasses PVC target discovery
	SecretDir       string   `json:"secret_dir,omitempty"` // Directory with accessKeyId and secretKey files (default /etc/secret)
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
	if input.Host == "" {
		return fmt.Errorf("host is required")
	}
	if input.VMName == "" {
		return fmt.Errorf("vm_name is required")
	}
	if input.Fingerprint == "" {
		return fmt.Errorf("fingerprint is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	slog.Info("NFC disk copy starting",
		"vm", input.VMName,
		"output_dir", input.OutputDir,
	)

	outputPath := input.OutputPath
	if outputPath == "" {
		outputPath = input.Workdir + "/copy-progress.json"
	}

	policy, err := v2vtls.CopyTLS(input.Insecure, input.CaCert)
	if err != nil {
		return err
	}

	lease, err := ExportVM(ctx, input.Host, input.Datacenter, policy, input.Fingerprint, input.VMName, input.SecretDir)
	if err != nil {
		return fmt.Errorf("NFC export: %w", err)
	}

	selected, err := FilterDiskURLs(lease.DiskURLs, input.SourceDisks)
	if err != nil {
		_ = lease.Abort(ctx)
		return fmt.Errorf("filter NFC disks: %w", err)
	}

	targets, err := resolveTargets(input.OutputDir, len(selected))
	if err != nil {
		_ = lease.Abort(ctx)
		return err
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
			if err := CopyDisk(ctx, lease, diskURL, target, func(pct int) {
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
			if err := types.WriteJSON(outputPath, progress); err != nil {
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

// resolveTargets returns the write targets. When outputDir is set, it creates
// file targets (disk0.img, disk1.img, …) in that directory. Otherwise it
// discovers PVC targets and filters to empty ones.
func resolveTargets(outputDir string, diskCount int) ([]Target, error) {
	if outputDir != "" {
		return FileTargets(outputDir, diskCount)
	}
	allTargets, err := DiscoverTargets()
	if err != nil {
		return nil, err
	}
	if err := logDiscoveredTargets(allTargets); err != nil {
		return nil, err
	}
	targets, err := EmptyTargets()
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("no empty PVC targets found")
	}
	return targets, nil
}

// FileTargets creates n file targets (disk0.img, disk1.img, …) in dir.
func FileTargets(dir string, n int) ([]Target, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}
	targets := make([]Target, n)
	for i := range targets {
		targets[i] = Target{
			Path:       filepath.Join(dir, fmt.Sprintf("disk%d.img", i)),
			IsBlockDev: false,
			Index:      i,
		}
	}
	slog.Info("file targets", "dir", dir, "count", n)
	return targets, nil
}
