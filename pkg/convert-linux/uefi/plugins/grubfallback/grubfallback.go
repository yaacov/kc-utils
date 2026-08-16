//go:build unix

package grubfallback

import (
	"log/slog"
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/common/uefi"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
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

// shimSiblingGrub returns the second-stage GRUB binary a shim loads from its
// own directory, or "" when the given binary is not a shim.
func shimSiblingGrub(binary string) string {
	switch binary {
	case "shimx64.efi", "shim.efi":
		return "grubx64.efi"
	case "shimaa64.efi":
		return "grubaa64.efi"
	default:
		return ""
	}
}

func (g *GrubFallback) ConvertToVirtio(guestRoot, espPath string) error {
	efiDir := filepath.Join(guestRoot, espPath, "EFI")
	fallbackDir := filepath.Join(efiDir, "BOOT")

	// Check whether a fallback already exists.
	for _, name := range []string{"bootx64.efi", "bootaa64.efi", "BOOTX64.EFI", "BOOTAA64.EFI"} {
		if guestio.FileExists(filepath.Join(fallbackDir, name)) {
			slog.Debug("GRUB fallback already exists, skipping", "path", filepath.Join(fallbackDir, name))
			return nil
		}
	}

	for _, c := range grubCandidates {
		src := filepath.Join(efiDir, c.subdir, c.binary)
		if !guestio.FileExists(src) {
			continue
		}
		if err := guestio.FileMkdirAll(fallbackDir, 0o755); err != nil {
			slog.Warn("failed to create BOOT dir", "error", err)
			return nil
		}
		dst := filepath.Join(fallbackDir, c.fallbackName)
		if err := guestio.FileCopy(src, dst); err != nil {
			slog.Warn("failed to copy GRUB fallback", "src", src, "dst", dst, "error", err)
			return nil
		}
		slog.Info("installed EFI fallback bootloader", "src", src, "dst", dst)

		// A shim loads its second-stage GRUB from the same directory it was
		// booted from. When the fallback binary is a shim, copy the sibling
		// GRUB binary into the fallback dir under its real name so a boot via
		// \EFI\BOOT\boot{x64,aa64}.efi (removable-media path) can chain into
		// GRUB. Without it the fallback loads shim, finds no GRUB, and fails.
		if grubName := shimSiblingGrub(c.binary); grubName != "" {
			grubSrc := filepath.Join(efiDir, c.subdir, grubName)
			grubDst := filepath.Join(fallbackDir, grubName)
			switch {
			case !guestio.FileExists(grubSrc):
				slog.Warn("shim fallback installed but sibling GRUB not found; fallback boot may fail", "expected", grubSrc)
			case guestio.FileExists(grubDst):
				slog.Debug("GRUB second stage already present beside fallback", "path", grubDst)
			default:
				if err := guestio.FileCopy(grubSrc, grubDst); err != nil {
					slog.Warn("failed to copy GRUB second stage for shim fallback", "src", grubSrc, "dst", grubDst, "error", err)
				} else {
					slog.Info("installed GRUB second stage beside shim fallback", "src", grubSrc, "dst", grubDst)
				}
			}
		}
		return nil
	}

	slog.Debug("no GRUB/shim EFI binary found for fallback copy")
	return nil
}
