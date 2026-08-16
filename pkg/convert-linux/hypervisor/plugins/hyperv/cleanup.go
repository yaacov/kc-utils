//go:build unix

package hyperv

import (
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor"
	"github.com/yaacov/kc-utils/pkg/convert-linux/systemd"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

type Cleanup struct{}

func init() {
	hypervisor.LinuxCleanups.Register("hyperv", &Cleanup{})
}

func (c *Cleanup) Detect(guestRoot string) bool {
	indicators := []string{
		filepath.Join(guestRoot, "usr", "sbin", "hv_kvp_daemon"),
		filepath.Join(guestRoot, "usr", "sbin", "hv_fcopy_daemon"),
		filepath.Join(guestRoot, "usr", "sbin", "hv_vss_daemon"),
	}
	for _, p := range indicators {
		if guestio.FileExists(p) {
			return true
		}
	}
	return false
}

func (c *Cleanup) Cleanup(guestRoot string) error {
	for _, unit := range []string{
		"hv-kvp-daemon.service",
		"hv-fcopy-daemon.service",
		"hv-vss-daemon.service",
		"hypervkvpd.service",
		"hypervfcopyd.service",
		"hypervvssd.service",
	} {
		systemd.DisableSystemdUnit(guestRoot, unit)
	}
	return nil
}
