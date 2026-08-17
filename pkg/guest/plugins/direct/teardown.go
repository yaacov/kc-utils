//go:build unix

package direct

import (
	"log/slog"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// Release frees process-local state (probe mounts) while keeping guest mounts,
// LUKS mappers, and loop devices in place for the convert stage.
func (b *Backend) Release() error {
	return b.UnmountProbes()
}

// ReleaseDevices closes LUKS mappers, deactivates LVM, and detaches loops.
func (b *Backend) ReleaseDevices() error {
	firstErr := b.CloseCryptMaps()
	b.CloseAllCryptMaps()
	if err := b.DeactivateLVM(); err != nil {
		slog.Warn("vgchange -an failed", "error", err)
	}
	for _, d := range b.disks {
		if err := d.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	b.disks = nil
	for _, path := range b.DiskPaths() {
		detachLoopsFor(path)
	}
	return firstErr
}

// Teardown fully unmounts and releases every resource this backend owns.
func (b *Backend) Teardown() error {
	firstErr := b.UnmountFilesystems()
	if err := b.ReleaseDevices(); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// TeardownDiscard is identical to Teardown for direct: writes go through live
// host mounts, so there is nothing extra to discard.
func (b *Backend) TeardownDiscard() error {
	return b.Teardown()
}

// TeardownMountRoot best-effort cleans orphaned direct-backend resources when
// no live backend handle is available.
func TeardownMountRoot(mountRoot string) error {
	unmountUnder(mountRoot)
	closeAllCryptMaps()
	if err := exec.Command("vgchange", "-an").Run(); err != nil {
		slog.Warn("vgchange -an failed", "error", err)
	}
	return nil
}

func unmountUnder(mountRoot string) {
	if mountRoot == "" {
		return
	}
	mountData, err := os.ReadFile("/proc/mounts")
	if err != nil {
		slog.Warn("reading /proc/mounts failed", "error", err)
		return
	}
	var mountPoints []string
	for line := range strings.SplitSeq(string(mountData), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		mp := fields[1]
		if mp == mountRoot || strings.HasPrefix(mp, mountRoot+"/") {
			mountPoints = append(mountPoints, mp)
		}
	}
	sort.Slice(mountPoints, func(i, j int) bool {
		return strings.Count(mountPoints[i], "/") > strings.Count(mountPoints[j], "/")
	})
	for _, mp := range mountPoints {
		if err := exec.Command("umount", mp).Run(); err != nil {
			slog.Warn("unmount failed", "mountpoint", mp, "error", err)
		}
	}
}

func detachLoopsFor(path string) {
	loOut, err := exec.Command("losetup", "-j", path).Output()
	if err != nil {
		return
	}
	for line := range strings.SplitSeq(string(loOut), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		loopDev := strings.TrimSpace(strings.SplitN(line, ":", 2)[0])
		if err := exec.Command("losetup", "-d", loopDev).Run(); err != nil {
			slog.Warn("detach loop device failed", "device", loopDev, "error", err)
		}
	}
}

func closeAllCryptMaps() {
	entries, err := os.ReadDir("/dev/mapper")
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.Name() == "control" {
			continue
		}
		_ = exec.Command("cryptsetup", "close", e.Name()).Run()
	}
}
