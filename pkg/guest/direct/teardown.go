//go:build unix

package direct

import (
	"log/slog"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/yaacov/kc-utils/pkg/guest/direct/luks"
	"github.com/yaacov/kc-utils/pkg/guest/direct/mount"
)

func (b *Backend) UnmountFilesystems() error {
	var firstErr error
	if b.probeMounts != nil {
		if err := b.probeMounts.UnmountAll(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if b.mounts != nil {
		if err := b.mounts.UnmountAll(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	unmountUnder(b.mountRoot)
	return firstErr
}

func (b *Backend) ReleaseDevices() error {
	var firstErr error
	for i := len(b.cryptMaps) - 1; i >= 0; i-- {
		if err := luks.Close(b.cryptMaps[i]); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	b.cryptMaps = nil
	closeAllCryptMaps()
	if err := exec.Command("vgchange", "-an").Run(); err != nil {
		slog.Warn("vgchange -an failed", "error", err)
	}
	for _, d := range b.disks {
		if err := d.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	b.disks = nil
	for _, path := range b.diskPaths {
		detachLoopsFor(path)
	}
	return firstErr
}

func (b *Backend) teardownResources() error {
	var firstErr error
	if err := b.UnmountFilesystems(); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := b.ReleaseDevices(); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
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
	for _, line := range strings.Split(string(mountData), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		mp := fields[1]
		if strings.HasPrefix(mp, mountRoot) {
			mountPoints = append(mountPoints, mp)
		}
	}
	sort.Slice(mountPoints, func(i, j int) bool {
		return strings.Count(mountPoints[i], "/") > strings.Count(mountPoints[j], "/")
	})
	for _, mp := range mountPoints {
		if err := mount.Unmount(mp); err != nil {
			slog.Warn("unmount failed", "mountpoint", mp, "error", err)
		}
	}
}

func detachLoopsFor(path string) {
	loOut, err := exec.Command("losetup", "-j", path).Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(loOut), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 1 {
			continue
		}
		loopDev := strings.TrimSpace(parts[0])
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
		name := e.Name()
		if name == "control" {
			continue
		}
		_ = luks.Close(name)
	}
}
