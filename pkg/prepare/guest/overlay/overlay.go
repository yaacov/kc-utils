package overlay

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Disk describes a disk used during in-place conversion.
type Disk struct {
	BackingPath string
	Path        string
	OverlayPath string
}

// Overlay tracks a qcow2 overlay for one disk.
type Overlay struct {
	Path        string
	BackingPath string
	Disk        *Disk
}

// CreateOverlays creates qcow2 overlay files in workdir and points disk.Path at each overlay.
func CreateOverlays(workdir string, disks []*Disk) ([]*Overlay, error) {
	var overlays []*Overlay
	for i, disk := range disks {
		if disk.BackingPath == "" {
			disk.BackingPath = disk.Path
		}
		overlayPath := filepath.Join(workdir, overlayFilename(disk.BackingPath, i)+".qcow2")
		slog.Info("creating qcow2 overlay",
			"index", i,
			"backing", disk.BackingPath,
			"overlay", overlayPath,
		)
		cmd := exec.Command("qemu-img", "create", "-f", "qcow2", "-b", disk.BackingPath, "-F", "raw", overlayPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			_ = os.Remove(overlayPath)
			DiscardOverlays(overlays)
			return nil, fmt.Errorf("create overlay for %s: %w", disk.BackingPath, err)
		}

		overlays = append(overlays, &Overlay{
			Path:        overlayPath,
			BackingPath: disk.BackingPath,
			Disk:        disk,
		})
		disk.OverlayPath = overlayPath
		disk.Path = overlayPath
	}
	slog.Info("qcow2 overlays ready", "count", len(overlays))
	return overlays, nil
}

// CommitOverlays merges overlays into backing disks.
func CommitOverlays(overlays []*Overlay) error {
	for _, o := range overlays {
		slog.Info("committing qcow2 overlay", "overlay", o.Path, "backing", o.BackingPath)
		cmd := exec.Command("qemu-img", "commit", o.Path)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			DiscardOverlays(overlays)
			return fmt.Errorf("commit overlay %s: %w", o.Path, err)
		}
		_ = os.Remove(o.Path)
	}
	restorePaths(overlays)
	slog.Info("qcow2 overlays committed", "count", len(overlays))
	return nil
}

// DiscardOverlays removes overlays without committing.
func DiscardOverlays(overlays []*Overlay) {
	if len(overlays) == 0 {
		return
	}
	slog.Info("discarding qcow2 overlays", "count", len(overlays))
	for _, o := range overlays {
		_ = os.Remove(o.Path)
	}
	restorePaths(overlays)
}

func restorePaths(overlays []*Overlay) {
	for _, o := range overlays {
		if o.Disk == nil {
			continue
		}
		o.Disk.Path = o.BackingPath
		o.Disk.OverlayPath = ""
	}
}

// RunWithOverlay wraps fn with overlay create/commit semantics.
func RunWithOverlay(workdir string, disks []*Disk, fn func() error) error {
	overlays, err := CreateOverlays(workdir, disks)
	if err != nil {
		return err
	}
	completed := false
	defer func() {
		if !completed {
			DiscardOverlays(overlays)
		}
	}()
	if err := fn(); err != nil {
		DiscardOverlays(overlays)
		completed = true
		return err
	}
	if err := CommitOverlays(overlays); err != nil {
		return err
	}
	completed = true
	return nil
}

func overlayFilename(backing string, index int) string {
	base := filepath.Base(backing)
	base = strings.NewReplacer("/", "_", ":", "_").Replace(base)
	if base == "" {
		base = "disk"
	}
	return fmt.Sprintf("overlay-%d-%s", index, base)
}
