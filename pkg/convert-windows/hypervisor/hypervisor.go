package hypervisor

import (
	"fmt"

	"github.com/yaacov/kc-utils/pkg/common/plugin"
	"github.com/yaacov/kc-utils/pkg/common/registry"
)

// WindowsRemove offline-removes hypervisor software from Windows guests.
// Detect/Remove receive both SYSTEM and SOFTWARE hives: uninstall keys live
// under SOFTWARE, while some cleanups (e.g. EC2) edit SYSTEM services.
type WindowsRemove interface {
	Detect(guestRoot string, systemHive, softwareHive registry.Hive) bool
	Remove(guestRoot string, systemHive, softwareHive registry.Hive) error
}

// WindowsServices disables hypervisor services on Windows guests.
// Detect receives the SYSTEM hive so it can check for service registry keys
// even when the hypervisor's install directory has already been removed by
// an earlier pipeline stage.
type WindowsServices interface {
	Detect(guestRoot string, systemHive registry.Hive, ccs string) bool
	ServiceNames() []string
	DisableServices(guestRoot string, hive registry.Hive, ccs string) error
}

var (
	WindowsRemoves          = plugin.NewRegistry[string, WindowsRemove]()
	WindowsServiceDisablers = plugin.NewRegistry[string, WindowsServices]()
)

// CurrentControlSet returns the active control set name from the SYSTEM hive.
func CurrentControlSet(systemHive registry.Hive) string {
	if n, err := systemHive.GetDWORD(`Select`, "Current"); err == nil && n > 0 {
		return fmt.Sprintf("ControlSet%03d", n)
	}
	return "ControlSet001"
}
