//go:build unix

package hyperv

import (
	"fmt"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor"
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

func (r *Remove) Remove(_ string, _ registry.Hive, _ registry.Hive) error {
	// Hyper-V integration components are inbox Windows drivers; they cannot be
	// safely removed offline and are left in place. Service disabling is handled
	// by the services plugin.
	return nil
}
