//go:build unix

package hyperv

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type Remove struct{}

func init() {
	hypervisor.WindowsRemoves.Register("hyperv", &Remove{})
}

func (r *Remove) Detect(_ string, systemHive registry.Hive, _ registry.Hive) bool {
	ccs := hypervisor.CurrentControlSet(systemHive)
	for _, svc := range []string{
		"vmicheartbeat", "vmicshutdown",
		"vmicvss", "vmictimesync", "vmicrdv",
		"vmicguestinterface", "vmickvpexchange",
		"vmicvmsession", "storflt",
	} {
		if systemHive.KeyExists(fmt.Sprintf("%s\\Services\\%s", ccs, svc)) {
			return true
		}
	}
	return false
}

func (r *Remove) Remove(guestRoot string, _ registry.Hive, _ registry.Hive) error {
	// Hyper-V integration components are inbox Windows drivers; they cannot be
	// safely removed offline. Service disabling is handled by the services plugin.
	for _, drv := range []string{
		"vmbus.sys", "storvsc.sys", "netvsc.sys",
		"VMBusHID.sys", "hypervideo.sys",
	} {
		p := filepath.Join(guestRoot, "Windows", "System32", "drivers", drv)
		if guest.FileExists(p) {
			slog.Info("found Hyper-V inbox driver, leaving in place", "driver", drv)
		}
	}
	return nil
}
