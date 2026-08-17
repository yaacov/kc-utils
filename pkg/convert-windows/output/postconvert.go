//go:build unix

package output

import (
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

// FixPermissions sets standard permissions on Guestfs firstboot files.
func FixPermissions(mountRoot string) {
	guestfsDir := filepath.Join(mountRoot, "Program Files", "Guestfs")
	if !guestio.FileExists(guestfsDir) {
		return
	}
	_ = guestio.FileWalkDir(guestfsDir, func(path string, isDir bool) error {
		if isDir {
			_ = guestio.FileChmod(path, 0o755)
		} else {
			_ = guestio.FileChmod(path, 0o644)
		}
		return nil
	})
}
