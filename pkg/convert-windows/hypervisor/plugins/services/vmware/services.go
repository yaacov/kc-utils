//go:build unix

package vmware

import (
	"fmt"
	"log/slog"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor"
)

type Services struct{}

func init() {
	hypervisor.WindowsServiceDisablers.Register("vmware", &Services{})
}

func (s *Services) Detect(_ string, systemHive registry.Hive, ccs string) bool {
	for _, svc := range s.ServiceNames() {
		svcPath := fmt.Sprintf("%s\\Services\\%s", ccs, svc)
		if systemHive.KeyExists(svcPath) {
			return true
		}
	}
	return false
}

func (s *Services) ServiceNames() []string {
	return []string{"VMTools", "VGAuthService", "VMwarePhysicalDiskHelper", "vm3dservice", "VMUSBArbService"}
}

func (s *Services) DisableServices(guestRoot string, hive registry.Hive, ccs string) error {
	for _, svc := range s.ServiceNames() {
		svcPath := fmt.Sprintf("%s\\Services\\%s", ccs, svc)
		if hive.KeyExists(svcPath) {
			hive.SetDWORD(svcPath, "Start", 4)
			slog.Info("disabled service", "service", svc)
		}
	}
	return nil
}
