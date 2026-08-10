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
		filepath.Join(guestRoot, "etc", "init.d", "ngt_guest_agent"),
	}
	for _, p := range indicators {
		if guest.FileExists(p) {
			return true
		}
	}
	return false
}

func (c *Cleanup) Cleanup(guestRoot string) error {
	for _, unit := range []string{
		"ngt_guest_agent.service",
		"ngt_self_service_restore.service",
		"nutanix-guest-agent.service",
	} {
		hypervisor.DisableSystemdUnit(guestRoot, unit)
	}
	_ = guest.FileRemove(filepath.Join(guestRoot, "etc", "rc.d", "init.d", "ngt_guest_agent"))
	_ = guest.FileRemove(filepath.Join(guestRoot, "etc", "init.d", "ngt_guest_agent"))
	hypervisor.RemovePaths(filepath.Join(guestRoot, "usr", "local", "nutanix", "ngt"))
	return nil
}
