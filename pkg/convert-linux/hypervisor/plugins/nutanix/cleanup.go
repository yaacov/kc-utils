//go:build linux

package nutanix

import (
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type Cleanup struct{}

func init() {
	hypervisor.LinuxCleanups.Register("nutanix", &Cleanup{})
}

func (c *Cleanup) Detect(guestRoot string) bool {
	indicators := []string{
		filepath.Join(guestRoot, "usr", "local", "nutanix", "ngt"),
		filepath.Join(guestRoot, "etc", "rc.d", "init.d", "ngt_guest_agent"),
	}
	for _, p := range indicators {
		if guest.FileExists(p) {
			return true
		}
	}
	return false
}

func (c *Cleanup) Cleanup(guestRoot string) error {
	services := []string{"ngt_guest_agent.service"}
	symlinkDirs := []string{
		filepath.Join(guestRoot, "etc", "systemd", "system", "multi-user.target.wants"),
		filepath.Join(guestRoot, "etc", "systemd", "system", "default.target.wants"),
	}
	for _, dir := range symlinkDirs {
		for _, svc := range services {
			_ = guest.FileRemove(filepath.Join(dir, svc))
		}
	}
	_ = guest.FileRemove(filepath.Join(guestRoot, "etc", "rc.d", "init.d", "ngt_guest_agent"))
	return nil
}
