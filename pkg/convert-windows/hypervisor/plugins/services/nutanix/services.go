//go:build unix

package nutanix

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

type Services struct{}

func init() {
	hypervisor.WindowsServiceDisablers.Register("nutanix", &Services{})
}

func (s *Services) Detect(guestRoot string, systemHive registry.Hive, ccs string) bool {
	for _, svc := range s.ServiceNames() {
		svcPath := fmt.Sprintf("%s\\Services\\%s", ccs, svc)
		if systemHive.KeyExists(svcPath) {
			return true
		}
	}
	indicators := []string{
		filepath.Join(guestRoot, "Program Files", "Nutanix"),
		filepath.Join(guestRoot, "Program Files (x86)", "Nutanix"),
	}
	for _, p := range indicators {
		if guestio.FileExists(p) {
			return true
		}
	}
	return false
}

func (s *Services) ServiceNames() []string {
	return []string{"NutanixGuestTools", "NutanixGuestAgent", "NgtService"}
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
