//go:build unix

package output

import (
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/guest"
)

// FixPermissions sets standard permissions on Guestfs firstboot files.
func FixPermissions(mountRoot string) {
	guestfsDir := filepath.Join(mountRoot, "Program Files", "Guestfs")
	if !guest.FileExists(guestfsDir) {
		return
	}
	_ = guest.FileWalkDir(guestfsDir, func(path string, isDir bool) error {
		if isDir {
			_ = guest.FileChmod(path, 0o755)
		} else {
			_ = guest.FileChmod(path, 0o644)
		}
		return nil
	})
}
