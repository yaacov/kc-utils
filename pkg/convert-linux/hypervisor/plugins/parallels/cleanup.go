//go:build linux

package parallels

import (
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type Cleanup struct{}

func init() {
	hypervisor.LinuxCleanups.Register("parallels", &Cleanup{})
}

func (c *Cleanup) Detect(guestRoot string) bool {
	return guest.FileExists(filepath.Join(guestRoot, "usr", "bin", "prlsrvctl")) ||
		guest.FileExists(filepath.Join(guestRoot, "usr", "sbin", "prltoolsd")) ||
		guest.FileExists(filepath.Join(guestRoot, "usr", "lib", "parallels-tools", "install"))
}

func (c *Cleanup) Cleanup(guestRoot string) error {
	hypervisor.DisableSystemdUnit(guestRoot, "prltoolsd.service")
	hypervisor.DisableSystemdUnit(guestRoot, "prl-xorg-cleanup.service")

	hypervisor.RemovePaths(
		filepath.Join(guestRoot, "usr", "lib", "parallels-tools"),
		filepath.Join(guestRoot, "usr", "lib64", "parallels-tools"),
	)
	return nil
}
