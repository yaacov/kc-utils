//go:build linux

package virtualbox

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type Services struct{}

func init() {
	hypervisor.WindowsServiceDisablers.Register("virtualbox", &Services{})
}

func (s *Services) Detect(guestRoot string, systemHive registry.Hive, ccs string) bool {
	for _, svc := range s.ServiceNames() {
		svcPath := fmt.Sprintf("%s\\Services\\%s", ccs, svc)
		if systemHive.KeyExists(svcPath) {
			return true
		}
	}
	p := filepath.Join(guestRoot, "Program Files", "Oracle", "VirtualBox Guest Additions")
	return guest.FileExists(p)
}

func (s *Services) ServiceNames() []string {
	return []string{"VBoxService", "VBoxGuest", "VBoxSF", "VBoxVideo", "VBoxMouse"}
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
