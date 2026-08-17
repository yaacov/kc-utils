//go:build unix

package parallels

import (
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor"
	"github.com/yaacov/kc-utils/pkg/convert-linux/systemd"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

type Cleanup struct{}

func init() {
	hypervisor.LinuxCleanups.Register("parallels", &Cleanup{})
}

func (c *Cleanup) Detect(guestRoot string) bool {
	return guestio.FileExists(filepath.Join(guestRoot, "usr", "bin", "prlsrvctl")) ||
		guestio.FileExists(filepath.Join(guestRoot, "usr", "sbin", "prltoolsd")) ||
		guestio.FileExists(filepath.Join(guestRoot, "usr", "lib", "parallels-tools", "install"))
}

func (c *Cleanup) Cleanup(guestRoot string) error {
	systemd.DisableSystemdUnit(guestRoot, "prltoolsd.service")
	systemd.DisableSystemdUnit(guestRoot, "prl-xorg-cleanup.service")
	systemd.DisableSystemdUnit(guestRoot, "prl-x11.service")

	systemd.RemovePaths(
		filepath.Join(guestRoot, "usr", "lib", "parallels-tools"),
		filepath.Join(guestRoot, "usr", "lib64", "parallels-tools"),
	)
	return nil
}
