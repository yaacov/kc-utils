//go:build unix

// Package direct mounts guest disks on the host via loop devices and the
// standard util-linux/LVM/cryptsetup tools. All domain logic lives in
// pkg/guest/core on top of a host-local runtime; this package adds only the
// host-specific parts: loop-device attachment and teardown.
package direct

import (
	"fmt"
	"log/slog"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest/core"
	"github.com/yaacov/kc-utils/pkg/guest/plugins/direct/disk"
	"github.com/yaacov/kc-utils/pkg/guest/runtime/local"
)

// Backend mounts guest disks on the host via mount(8) and requires CAP_SYS_ADMIN.
// It embeds the shared core backend (running on a host-local runtime) and owns
// only the loop devices it attaches.
type Backend struct {
	*core.Backend
	mountRoot string
	disks     []*disk.DiskSetup
}

// New returns an unconfigured direct backend bound to a host-local runtime.
func New() *Backend {
	return &Backend{Backend: core.New(local.New(), true)}
}

// NewMounted returns a direct backend that adopts an already-mounted tree
// (used by Attach for the convert/finalize handoff, which does not re-scan).
func NewMounted(disks []types.DiskSpec, mountRoot string, diskInfos []types.DiskInfo) *Backend {
	b := New()
	b.mountRoot = mountRoot
	b.SetGuestRoot(mountRoot)
	b.AdoptDisks(diskInfos, diskSpecPaths(disks))
	return b
}

// Setup attaches each disk via a loop device, enumerates partitions with lsblk
// (in core), then activates any LVM volume groups.
func (b *Backend) Setup(disks []types.DiskSpec, mountRoot string) error {
	b.mountRoot = mountRoot
	b.SetGuestRoot(mountRoot)
	for _, d := range disks {
		slog.Info("setting up disk", "path", d.Path, "format", d.Format)
		ds, err := disk.Setup(d.Path)
		if err != nil {
			_ = b.Teardown()
			return fmt.Errorf("setting up disk %s: %w", d.Path, err)
		}
		b.disks = append(b.disks, ds)
		if err := b.DiscoverDevice(ds.LoopDevice, d.Path, d.Format); err != nil {
			_ = b.Teardown()
			return fmt.Errorf("discovering disk %s: %w", d.Path, err)
		}
	}
	if err := b.ScanLVM(); err != nil {
		slog.Warn("LVM scan failed", "error", err)
	}
	return nil
}

// host translates a guest-absolute path to its host path under mountRoot. It is
// kept as a package method for LiveHostPath and the direct-package tests.
func (b *Backend) host(guestPath string) string { return b.HostPath(guestPath) }

// LiveHostPath reports that direct exposes a real host mount tree, so callers
// can operate on files in place instead of downloading them.
func (b *Backend) LiveHostPath(guestPath string) (string, bool) {
	return b.host(guestPath), true
}

// Sync is a no-op for direct: edits go through live host mounts.
func (b *Backend) Sync() error { return nil }

func diskSpecPaths(disks []types.DiskSpec) []string {
	paths := make([]string, 0, len(disks))
	for _, d := range disks {
		paths = append(paths, d.Path)
	}
	return paths
}
