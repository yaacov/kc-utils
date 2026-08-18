//go:build unix

package disk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
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

	mode := info.Mode()
	switch {
	case mode&os.ModeCharDevice != 0:
		return nil, fmt.Errorf("%s: character devices are not supported", path)
	case mode&os.ModeDevice != 0:
		// block device
	case mode.IsRegular():
		// disk image file for losetup
	default:
		return nil, fmt.Errorf("%s: path must be a block device or regular file", path)
	}

	ds := &DiskSetup{Path: path}

	if mode&os.ModeDevice != 0 {
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

	type part struct {
		index int
		path  string
	}
	var parts []part
	for _, m := range matches {
		if m == ds.LoopDevice {
			continue
		}
		idx := partitionIndex(m, ds.LoopDevice)
		if idx <= 0 {
			continue
		}
		parts = append(parts, part{index: idx, path: m})
	}
	sort.Slice(parts, func(i, j int) bool { return parts[i].index < parts[j].index })
	for _, p := range parts {
		ds.Partitions = append(ds.Partitions, PartitionDevice{
			Index:      p.index,
			DevicePath: p.path,
		})
	}
	return nil
}

func partitionIndex(devicePath, loopDevice string) int {
	suffix := strings.TrimPrefix(devicePath, loopDevice)
	if suffix == "" || suffix == devicePath {
		return 0
	}
	if strings.HasPrefix(suffix, "p") && len(suffix) > 1 {
		suffix = suffix[1:]
	}
	n, err := strconv.Atoi(suffix)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// Close detaches the loop device if one was created.
func (ds *DiskSetup) Close() error {
	if !ds.IsLoop {
		return nil
	}
	return exec.Command("losetup", "-d", ds.LoopDevice).Run()
}
