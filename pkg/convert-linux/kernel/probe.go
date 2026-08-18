//go:build unix

package kernel

import (
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/guest"
)

// ModulesDir returns the guest's kernel modules directory. Some distros and
// test fixtures place modules under /usr/lib/modules instead of /lib/modules.
func ModulesDir(guestRoot string) string {
	candidates := []string{
		filepath.Join(guestRoot, "lib", "modules"),
		filepath.Join(guestRoot, "usr", "lib", "modules"),
	}
	for _, dir := range candidates {
		if guest.FileIsDir(dir) {
			return dir
		}
	}
	return candidates[0]
}

// ProbeModules checks whether a kernel version has virtio support and whether
// it is a Xen-PV-only kernel (has xen block drivers but no virtio drivers).
// A single glob replaces per-directory ReadDir calls to avoid destabilising
// the guestfish daemon (see initramfs/virtio.go for rationale).
func ProbeModules(guestRoot, version string) (hasVirtio, isXenPV bool) {
	driversDir := filepath.Join(ModulesDir(guestRoot), version, "kernel", "drivers")
	if !guest.FileIsDir(driversDir) {
		return false, false
	}
	matches, err := guest.FileGlob(filepath.Join(driversDir, "*", "*.ko*"))
	if err != nil {
		slog.Warn("kernel module glob failed", "dir", driversDir, "version", version, "error", err)
		return false, false
	}
	for _, m := range matches {
		name := modBaseName(filepath.Base(m))
		switch name {
		case "virtio_blk", "virtio_net", "virtio_pci":
			hasVirtio = true
		case "xen-blkfront":
			isXenPV = true
		}
	}
	if hasVirtio {
		isXenPV = false
	}
	return
}

// modBaseName strips the .ko suffix and any compression extension
// (e.g. "virtio_blk.ko.xz" -> "virtio_blk").
func modBaseName(filename string) string {
	if name, _, ok := strings.Cut(filename, ".ko"); ok {
		return name
	}
	return filename
}
