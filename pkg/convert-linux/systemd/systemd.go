//go:build unix

// Package systemd provides shared stage-local helpers for guest systemd unit
// management during Linux conversion. Not a pipeline block — used by hypervisor
// cleanup plugins and network detection.
package systemd

import (
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/guest"
)

// UnitMaskTarget is the guest-absolute path DisableSystemdUnit uses when masking a unit.
const UnitMaskTarget = "/dev/null"

// SystemdUnitMaskPath returns the host path to the mask symlink DisableSystemdUnit creates.
func SystemdUnitMaskPath(guestRoot, unit string) string {
	return filepath.Join(guestRoot, "etc", "systemd", "system", unit)
}

// VendorWantsPath returns the host path to a vendor preset wants symlink for a unit.
func VendorWantsPath(guestRoot, unit string) string {
	return filepath.Join(guestRoot, "usr", "lib", "systemd", "system", "multi-user.target.wants", unit)
}

// wantsRelDirs lists the guest-relative wants directories DisableSystemdUnit
// and DisableEC2NetHooks scan, covering both /etc and vendor (/usr/lib)
// presets across the target types units commonly enable under.
var wantsRelDirs = []string{
	filepath.Join("etc", "systemd", "system", "multi-user.target.wants"),
	filepath.Join("etc", "systemd", "system", "default.target.wants"),
	filepath.Join("etc", "systemd", "system", "sockets.target.wants"),
	filepath.Join("etc", "systemd", "system", "graphical.target.wants"),
	filepath.Join("usr", "lib", "systemd", "system", "multi-user.target.wants"),
	filepath.Join("usr", "lib", "systemd", "system", "default.target.wants"),
	filepath.Join("usr", "lib", "systemd", "system", "sockets.target.wants"),
	filepath.Join("usr", "lib", "systemd", "system", "graphical.target.wants"),
}

// DisableSystemdUnit removes wants symlinks and masks the unit under the guest root.
func DisableSystemdUnit(guestRoot string, unit string) {
	for _, rel := range wantsRelDirs {
		_ = guest.FileRemove(filepath.Join(guestRoot, rel, unit))
	}
	maskPath := filepath.Join(guestRoot, "etc", "systemd", "system", unit)
	if err := guest.FileMkdirAll(filepath.Dir(maskPath), 0o755); err != nil {
		slog.Warn("creating systemd unit mask dir failed", "path", filepath.Dir(maskPath), "unit", unit, "error", err)
	}
	_ = guest.FileRemove(maskPath)
	if err := guest.FileSymlink("/dev/null", maskPath); err != nil {
		slog.Warn("masking systemd unit failed", "path", maskPath, "unit", unit, "error", err)
	}
}

// RemovePaths removes files/directories if present.
func RemovePaths(paths ...string) {
	for _, p := range paths {
		_ = guest.FileRemoveAll(p)
	}
}

// UnitWantsEnabled reports whether a unit has a wants symlink under standard
// targets. The symlink entry itself is checked via readlink rather than
// guest.FileExists, since FileExists follows the link (os.Stat) and would
// report false for a real absolute-target symlink whose target does not
// resolve under guestRoot.
func UnitWantsEnabled(guestRoot, unit string) bool {
	for _, rel := range wantsRelDirs {
		path := filepath.Join(guestRoot, rel, unit)
		if _, err := guest.FileReadlink(path); err == nil {
			return true
		}
		if guest.FileExists(path) {
			return true
		}
	}
	return false
}

// UnitIsMasked reports whether unit is masked to /dev/null under etc/systemd/system.
func UnitIsMasked(guestRoot, unit string) bool {
	maskPath := SystemdUnitMaskPath(guestRoot, unit)
	if !guest.FileExists(maskPath) {
		return false
	}
	target, err := guest.FileReadlink(maskPath)
	return err == nil && target == UnitMaskTarget
}

// DisableEC2NetHooks masks EC2-only networking units that depend on IMDS or ENI policy routing.
func DisableEC2NetHooks(guestRoot string) {
	DisableSystemdUnit(guestRoot, "set-hostname-imds.service")

	seen := make(map[string]bool)
	for _, rel := range wantsRelDirs {
		entries, err := guest.FileReadDir(filepath.Join(guestRoot, rel))
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name
			if seen[name] {
				continue
			}
			if strings.HasPrefix(name, "policy-routes@") ||
				strings.HasPrefix(name, "refresh-policy-routes@") {
				seen[name] = true
				DisableSystemdUnit(guestRoot, name)
			}
		}
	}
}
