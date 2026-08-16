//go:build unix

package bcdeditor

import (
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/common/uefi"
	"github.com/yaacov/kc-utils/pkg/guest"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

const (
	bootManagerGUID            = `{9dea862c-5cdd-4e70-acc1-f32b344d4795}`
	bcdDefaultObjectElement    = `23000003`
	bcdGraphicsDisabledElement = `16000046`
)

type BCDEditor struct{}

func init() {
	uefi.Editors.Register("bcd", &BCDEditor{})
}

func (b *BCDEditor) ConvertToVirtio(guestRoot, espPath string) error {
	efiDir := filepath.Join(guestRoot, espPath, "EFI")

	// Remove graphicsmodedisabled from BCD hive if present.
	bcdPath := filepath.Join(efiDir, "Microsoft", "Boot", "BCD")
	if guestio.FileExists(bcdPath) {
		b.removeGraphicsDisabled(bcdPath)
	}

	// Ensure the fallback boot directory exists.
	fallbackDir := filepath.Join(efiDir, "Boot")
	if err := guestio.FileMkdirAll(fallbackDir, 0o755); err != nil {
		slog.Warn("failed to create fallback boot dir", "error", err)
		return nil
	}

	// Try to copy the Microsoft bootloader as the fallback EFI binary.
	// Try both x64 and aa64 variants.
	srcPath := filepath.Join(efiDir, "Microsoft", "Boot", "bootmgfw.efi")
	if !guestio.FileExists(srcPath) {
		slog.Warn("Microsoft bootloader not found, skipping fallback copy", "path", srcPath)
		return nil
	}

	// Determine fallback filename: try bootx64.efi first, then bootaa64.efi.
	fallbackName := "bootx64.efi"
	fallbackPath := filepath.Join(fallbackDir, fallbackName)

	// If x64 fallback already exists, check for aa64.
	if guestio.FileExists(fallbackPath) {
		slog.Debug("fallback bootloader already exists, skipping", "path", fallbackPath)
		return nil
	}

	// Also check aa64 variant.
	aa64Path := filepath.Join(fallbackDir, "bootaa64.efi")
	if guestio.FileExists(aa64Path) {
		slog.Debug("fallback bootloader already exists, skipping", "path", aa64Path)
		return nil
	}

	slog.Debug("copying bootloader", "src", srcPath, "dst", fallbackPath)
	if err := guestio.FileCopy(srcPath, fallbackPath); err != nil {
		slog.Warn("failed to copy fallback bootloader", "error", err)
		// Best-effort: don't fail the conversion.
		return nil
	}

	slog.Info("fallback EFI bootloader installed", "path", fallbackPath)
	return nil
}

func (b *BCDEditor) removeGraphicsDisabled(bcdPath string) {
	editor, ok := registry.Editors.Get("hivex")
	if !ok {
		slog.Warn("hivex editor not registered, skipping BCD edit")
		return
	}

	hostPath := bcdPath
	guestPath := ""
	saved := false
	g := guest.Active()
	if g != nil {
		rel, err := filepath.Rel(g.RootPath(), bcdPath)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			guestPath = "/" + filepath.ToSlash(rel)
			checkedOut, err := g.Checkout(guestPath)
			if err != nil {
				slog.Warn("checkout BCD hive", "error", err)
				return
			}
			hostPath = checkedOut
			defer func() {
				if !saved {
					g.DiscardCheckout(hostPath)
					return
				}
				if err := g.Checkin(guestPath, hostPath); err != nil {
					slog.Warn("checkin BCD hive", "error", err)
					g.DiscardCheckout(hostPath)
				}
			}()
		}
	}

	hive, err := editor.OpenHive(hostPath)
	if err != nil {
		slog.Warn("opening BCD hive", "error", err)
		return
	}
	defer hive.Close()

	defaultEntryPath := `Objects\` + bootManagerGUID + `\Elements\` + bcdDefaultObjectElement
	guid, err := hive.GetString(defaultEntryPath, "Element")
	if err != nil {
		slog.Warn("reading default boot entry", "error", err)
		return
	}

	graphicsPath := `Objects\` + guid + `\Elements\` + bcdGraphicsDisabledElement
	if !hive.KeyExists(graphicsPath) {
		slog.Debug("graphicsmodedisabled not set, nothing to remove")
		return
	}

	hive.DeleteKey(graphicsPath)
	if guestPath != "" && g != nil {
		if err := g.MergeHive(guestPath, hive.PendingReg()); err != nil {
			slog.Warn("saving BCD hive", "error", err)
			return
		}
		slog.Info("removed graphicsmodedisabled from BCD")
		return
	}
	if err := hive.Save(); err != nil {
		slog.Warn("saving BCD hive", "error", err)
		return
	}
	saved = true
	slog.Info("removed graphicsmodedisabled from BCD")
}
