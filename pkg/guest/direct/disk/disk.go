//go:build linux

package disk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DiskSetup represents a disk that has been set up for access.
type DiskSetup struct {
	Path       string
	IsLoop     bool
	LoopDevice string
	Partitions []PartitionDevice
}

// PartitionDevice represents a partition exposed by the kernel.
type PartitionDevice struct {
	Index      int
	DevicePath string
}

// Setup prepares a disk for access. For block devices, exposes partitions
// directly. For regular files, creates a loop device first.
func Setup(path string) (*DiskSetup, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	ds := &DiskSetup{Path: path}

	if info.Mode()&os.ModeDevice != 0 {
		ds.LoopDevice = path
		if err := ds.scanPartitions(); err != nil {
			return nil, err
		}
		return ds, nil
	}

	out, err := exec.Command("losetup", "--partscan", "--find", "--show", path).Output()
	if err != nil {
		return nil, fmt.Errorf("losetup %s: %w", path, err)
	}
	ds.IsLoop = true
	ds.LoopDevice = strings.TrimSpace(string(out))

	time.Sleep(200 * time.Millisecond)
	if err := ds.scanPartitions(); err != nil {
		ds.Close()
		return nil, err
	}
	return ds, nil
}

func (ds *DiskSetup) scanPartitions() error {
	pattern := ds.LoopDevice + "p*"
	if strings.Contains(ds.LoopDevice, "sd") || strings.Contains(ds.LoopDevice, "vd") || strings.Contains(ds.LoopDevice, "nvme") {
		if strings.Contains(ds.LoopDevice, "nvme") {
			pattern = ds.LoopDevice + "p*"
		} else {
			base := filepath.Base(ds.LoopDevice)
			dir := filepath.Dir(ds.LoopDevice)
			pattern = filepath.Join(dir, base+"[0-9]*")
		}
	}

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	// Fallback for containers where losetup --partscan doesn't create device nodes
	if len(matches) == 0 && ds.IsLoop {
		_ = exec.Command("partx", "--add", ds.LoopDevice).Run()
		time.Sleep(200 * time.Millisecond)
		matches, _ = filepath.Glob(pattern)
	}

	for i, m := range matches {
		if m == ds.LoopDevice {
			continue
		}
		ds.Partitions = append(ds.Partitions, PartitionDevice{
			Index:      i + 1,
			DevicePath: m,
		})
	}
	return nil
}

// Close detaches the loop device if one was created.
func (ds *DiskSetup) Close() error {
	if !ds.IsLoop {
		return nil
	}
	return exec.Command("losetup", "-d", ds.LoopDevice).Run()
}
