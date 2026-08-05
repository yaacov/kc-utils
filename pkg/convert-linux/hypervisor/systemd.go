//go:build linux

package hypervisor

import (
	"github.com/yaacov/kc-utils/pkg/guest"
	"path/filepath"
)

// DisableSystemdUnit removes wants symlinks and masks the unit under the guest root.
func DisableSystemdUnit(guestRoot string, unit string) {
	wants := []string{
		filepath.Join(guestRoot, "etc", "systemd", "system", "multi-user.target.wants", unit),
		filepath.Join(guestRoot, "etc", "systemd", "system", "default.target.wants", unit),
		filepath.Join(guestRoot, "usr", "lib", "systemd", "system", "multi-user.target.wants", unit),
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
