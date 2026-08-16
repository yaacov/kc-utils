//go:build unix

package disk

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// DiskSetup represents a disk that has been attached for access. Partition
// enumeration is done by the core backend via lsblk; this package only owns
// loop-device attachment/detachment.
type DiskSetup struct {
	Path       string
	IsLoop     bool
	LoopDevice string
}

// Setup prepares a disk for access. Block devices are used directly; regular
// files are attached to a loop device with kernel partition scanning enabled.
// The returned DiskSetup's LoopDevice is the block device to enumerate.
func Setup(path string) (*DiskSetup, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	ds := &DiskSetup{Path: path}

	if info.Mode()&os.ModeDevice != 0 {
		ds.LoopDevice = path
		ensurePartitions(ds.LoopDevice)
		return ds, nil
	}

	out, err := exec.Command("losetup", "--partscan", "--find", "--show", path).Output()
	if err != nil {
		return nil, fmt.Errorf("losetup %s: %w", path, err)
	}
	ds.IsLoop = true
	ds.LoopDevice = strings.TrimSpace(string(out))

	// Give udev/the kernel a moment to materialize partition nodes.
	time.Sleep(200 * time.Millisecond)
	ensurePartitions(ds.LoopDevice)
	return ds, nil
}

// ensurePartitions best-effort asks the kernel to (re)read the partition table
// so /dev/<disk>pN nodes exist. losetup --partscan usually does this already;
// partx -a is a fallback for containers where udev is absent. Both are safe to
// run when partitions already exist, so errors are ignored.
func ensurePartitions(device string) {
	_ = exec.Command("partx", "-a", device).Run()
	time.Sleep(200 * time.Millisecond)
}

// Close detaches the loop device if one was created.
func (ds *DiskSetup) Close() error {
	if !ds.IsLoop {
		return nil
	}
	return exec.Command("losetup", "-d", ds.LoopDevice).Run()
}
