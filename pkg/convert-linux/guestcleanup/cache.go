//go:build unix

package guestcleanup

import (
	"github.com/yaacov/kc-utils/pkg/guest"
	"log/slog"
	"os"
	"path/filepath"
)

// Clean removes stale blkid and LVM caches that reference old device names,
// and stale RPM DB BDB lock files that can block firstboot package installs.
func Clean(guestRoot string) {
	cacheFiles := []string{
		filepath.Join(guestRoot, "etc", "blkid.tab"),
		filepath.Join(guestRoot, "run", "blkid", "blkid.tab"),
		filepath.Join(guestRoot, "etc", "lvm", "cache", ".cache"),
	}
	for _, f := range cacheFiles {
		if err := guest.FileRemove(f); err != nil && !os.IsNotExist(err) {
			slog.Warn("removing cache failed", "path", f, "error", err)
		}
	}

	// Remove stale RPM DB lock files (RHBZ#1143866). These are always
	// stale in an offline-mounted guest and can cause rpm/dnf to hang
	// or fail during firstboot package installation.
	rpmLocks, _ := guest.FileGlob(filepath.Join(guestRoot, "var", "lib", "rpm", "__db.00*"))
	for _, f := range rpmLocks {
		if err := guest.FileRemove(f); err != nil && !os.IsNotExist(err) {
			slog.Warn("removing RPM DB lock failed", "path", f, "error", err)
		}
	}
}
