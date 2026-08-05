//go:build linux

package bootconfig

import (
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yaacov/kc-utils/pkg/guest"
)

var (
	driverRe     = regexp.MustCompile(`(?im)^(\s*Driver\s+)"[^"]*"`)
	vendorNameRe = regexp.MustCompile(`(?im)^\s*VendorName\s+"[^"]*"\s*\n`)
	boardNameRe  = regexp.MustCompile(`(?im)^\s*BoardName\s+"[^"]*"\s*\n`)

	// deviceSectionRe matches Section "Device" ... EndSection blocks.
	deviceSectionRe = regexp.MustCompile(`(?is)(Section\s+"Device".*?EndSection)`)
)

// ConfigureXorgDriver sets the X.org display driver to modesetting in
// xorg.conf or XF86Config if present. This matches virt-v2v behavior
// for VMware-origin guests that ship an xorg.conf with a vendor-specific
// driver (e.g. vmwgfx).
func ConfigureXorgDriver(guestRoot string) {
	xorgConf := filepath.Join(guestRoot, "etc", "X11", "xorg.conf")
	if !guest.FileExists(xorgConf) {
		alt := filepath.Join(guestRoot, "etc", "X11", "XF86Config")
		if !guest.FileExists(alt) {
			return
		}
		xorgConf = alt
	}

	data, err := guest.FileRead(xorgConf)
	if err != nil {
		slog.Warn("reading xorg config failed", "path", xorgConf, "error", err)
		return
	}

	content := string(data)
	updated := false

	content = deviceSectionRe.ReplaceAllStringFunc(content, func(section string) string {
		orig := section

		section = driverRe.ReplaceAllString(section, `${1}"modesetting"`)
		section = vendorNameRe.ReplaceAllString(section, "")
		section = boardNameRe.ReplaceAllString(section, "")

		if section != orig {
			updated = true
		}
		return section
	})

	if !updated {
		return
	}

	// Collapse any double blank lines left after removing VendorName/BoardName.
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}

	if err := guest.FileWrite(xorgConf, []byte(content), 0o644); err != nil {
		slog.Warn("writing xorg config failed", "path", xorgConf, "error", err)
	} else {
		slog.Info("updated xorg display driver to modesetting", "path", xorgConf)
	}
}
