//go:build unix

package grubfallback

import (
	"log/slog"
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/common/uefi"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type GrubFallback struct{}

func init() {
	uefi.Editors.Register("grub-fallback", &GrubFallback{})
}

// grubCandidates lists the EFI binaries to try as the fallback bootloader,
// in preference order. These cover RHEL/CentOS (shimx64.efi, grubx64.efi),
// Debian/Ubuntu (shimx64.efi, grubx64.efi), SUSE (shim.efi, grubx64.efi),
// and ARM64 variants.
var grubCandidates = []struct {
	subdir       string
	binary       string
	fallbackName string
}{
	{"redhat", "shimx64.efi", "bootx64.efi"},
	{"redhat", "grubx64.efi", "bootx64.efi"},
	{"centos", "shimx64.efi", "bootx64.efi"},
	{"ubuntu", "shimx64.efi", "bootx64.efi"},
	{"ubuntu", "grubx64.efi", "bootx64.efi"},
	{"debian", "grubx64.efi", "bootx64.efi"},
	{"suse", "shim.efi", "bootx64.efi"},
	{"suse", "grubx64.efi", "bootx64.efi"},
	{"fedora", "shimx64.efi", "bootx64.efi"},
	{"fedora", "grubx64.efi", "bootx64.efi"},
	{"redhat", "shimaa64.efi", "bootaa64.efi"},
	{"redhat", "grubaa64.efi", "bootaa64.efi"},
}

func (g *GrubFallback) ConvertToVirtio(guestRoot, espPath string) error {
	efiDir := filepath.Join(guestRoot, espPath, "EFI")
	fallbackDir := filepath.Join(efiDir, "BOOT")

	// Check whether a fallback already exists.
	for _, name := range []string{"bootx64.efi", "bootaa64.efi", "BOOTX64.EFI", "BOOTAA64.EFI"} {
		if guest.FileExists(filepath.Join(fallbackDir, name)) {
			slog.Debug("GRUB fallback already exists, skipping", "path", filepath.Join(fallbackDir, name))
			return nil
		}
	}

	for _, c := range grubCandidates {
		src := filepath.Join(efiDir, c.subdir, c.binary)
		if !guest.FileExists(src) {
			continue
		}
		if err := guest.FileMkdirAll(fallbackDir, 0o755); err != nil {
			slog.Warn("failed to create BOOT dir", "error", err)
			return nil
		}
		dst := filepath.Join(fallbackDir, c.fallbackName)
		if err := guest.FileCopy(src, dst); err != nil {
			slog.Warn("failed to copy GRUB fallback", "src", src, "dst", dst, "error", err)
			return nil
		}
		slog.Info("installed EFI fallback bootloader", "src", src, "dst", dst)
		return nil
	}

	slog.Debug("no GRUB/shim EFI binary found for fallback copy")
	return nil
}
