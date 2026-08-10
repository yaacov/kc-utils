//go:build linux

package hypervisor

import (
	"github.com/yaacov/kc-utils/pkg/guest"
	"path/filepath"
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

// DisableSystemdUnit removes wants symlinks and masks the unit under the guest root.
func DisableSystemdUnit(guestRoot string, unit string) {
	wants := []string{
		filepath.Join(guestRoot, "etc", "systemd", "system", "multi-user.target.wants", unit),
		filepath.Join(guestRoot, "etc", "systemd", "system", "default.target.wants", unit),
		filepath.Join(guestRoot, "etc", "systemd", "system", "sockets.target.wants", unit),
		filepath.Join(guestRoot, "etc", "systemd", "system", "graphical.target.wants", unit),
		filepath.Join(guestRoot, "usr", "lib", "systemd", "system", "multi-user.target.wants", unit),
		filepath.Join(guestRoot, "usr", "lib", "systemd", "system", "default.target.wants", unit),
		filepath.Join(guestRoot, "usr", "lib", "systemd", "system", "sockets.target.wants", unit),
		filepath.Join(guestRoot, "usr", "lib", "systemd", "system", "graphical.target.wants", unit),
	}
	for _, p := range wants {
		_ = guest.FileRemove(p)
	}
	maskPath := filepath.Join(guestRoot, "etc", "systemd", "system", unit)
	_ = guest.FileMkdirAll(filepath.Dir(maskPath), 0o755)
	_ = guest.FileRemove(maskPath)
	_ = guest.FileSymlink("/dev/null", maskPath)
}

// RemovePaths removes files/directories if present.
func RemovePaths(paths ...string) {
	for _, p := range paths {
		_ = guest.FileRemoveAll(p)
	}
}
