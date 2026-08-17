//go:build unix

// Package systemd provides shared stage-local helpers for guest systemd unit
// management during Linux conversion. Not a pipeline block — used by hypervisor
// cleanup plugins and network detection.
package systemd

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/guest/guestio"
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

var unitFileRelDirs = []string{
	filepath.Join("etc", "systemd", "system"),
	filepath.Join("usr", "lib", "systemd", "system"),
	filepath.Join("lib", "systemd", "system"),
}

// DisableSystemdUnit removes wants symlinks and masks the unit under the guest root.
func DisableSystemdUnit(guestRoot string, unit string) {
	for _, rel := range wantsRelDirs {
		_ = guestio.FileRemove(filepath.Join(guestRoot, rel, unit))
	}
	maskPath := filepath.Join(guestRoot, "etc", "systemd", "system", unit)
	if err := guestio.FileMkdirAll(filepath.Dir(maskPath), 0o755); err != nil {
		slog.Warn("creating systemd unit mask dir failed", "path", filepath.Dir(maskPath), "unit", unit, "error", err)
	}
	_ = guestio.FileRemove(maskPath)
	if err := guestio.FileSymlink("/dev/null", maskPath); err != nil {
		slog.Warn("masking systemd unit failed", "path", maskPath, "unit", unit, "error", err)
	}
}

// RemovePaths removes files/directories if present.
func RemovePaths(paths ...string) {
	for _, p := range paths {
		_ = guestio.FileRemoveAll(p)
	}
}

// UnitWantsEnabled reports whether a unit has a wants symlink under standard
// targets. The symlink entry itself is checked via readlink rather than
// guestio.FileExists, since FileExists follows the link (os.Stat) and would
// report false for a real absolute-target symlink whose target does not
// resolve under guestRoot.
func UnitWantsEnabled(guestRoot, unit string) bool {
	for _, rel := range wantsRelDirs {
		path := filepath.Join(guestRoot, rel, unit)
		if _, err := guestio.FileReadlink(path); err == nil {
			return true
		}
		if guestio.FileExists(path) {
			return true
		}
	}
	return false
}

// EnableSystemdUnit unmasks unit if needed and creates an admin wants symlink
// so the unit starts on next boot. Returns an error when the unit file is missing.
func EnableSystemdUnit(guestRoot, unit string) error {
	if UnitIsMasked(guestRoot, unit) {
		if err := guestio.FileRemove(SystemdUnitMaskPath(guestRoot, unit)); err != nil {
			return fmt.Errorf("unmasking %s: %w", unit, err)
		}
	}
	if UnitWantsEnabled(guestRoot, unit) {
		return nil
	}

	unitPath, err := findUnitFileGuestPath(guestRoot, unit)
	if err != nil {
		return err
	}

	wantsDir := filepath.Join(guestRoot, "etc", "systemd", "system", "multi-user.target.wants")
	if err := guestio.FileMkdirAll(wantsDir, 0o755); err != nil {
		return fmt.Errorf("creating wants dir for %s: %w", unit, err)
	}
	wantsLink := filepath.Join(wantsDir, unit)
	_ = guestio.FileRemove(wantsLink)
	if err := guestio.FileSymlink(unitPath, wantsLink); err != nil {
		return fmt.Errorf("enabling %s: %w", unit, err)
	}
	return nil
}

func findUnitFileGuestPath(guestRoot, unit string) (string, error) {
	for _, rel := range unitFileRelDirs {
		hostPath := filepath.Join(guestRoot, rel, unit)
		if guestio.FileExists(hostPath) {
			return "/" + filepath.ToSlash(filepath.Join(rel, unit)), nil
		}
	}
	return "", fmt.Errorf("unit file not found: %s", unit)
}

// UnitIsMasked reports whether unit is masked to /dev/null under etc/systemd/system.
func UnitIsMasked(guestRoot, unit string) bool {
	maskPath := SystemdUnitMaskPath(guestRoot, unit)
	if !guestio.FileExists(maskPath) {
		return false
	}
	target, err := guestio.FileReadlink(maskPath)
	return err == nil && target == UnitMaskTarget
}

// DisableEC2NetHooks masks EC2-only networking units that depend on IMDS or ENI policy routing.
func DisableEC2NetHooks(guestRoot string) {
	DisableSystemdUnit(guestRoot, "set-hostname-imds.service")

	seen := make(map[string]bool)
	for _, rel := range wantsRelDirs {
		entries, err := guestio.FileReadDir(filepath.Join(guestRoot, rel))
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
