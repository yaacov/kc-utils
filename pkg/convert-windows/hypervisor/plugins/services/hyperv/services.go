//go:build unix

package hyperv

import (
	"fmt"
	"log/slog"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor"
)

type Services struct{}

func init() {
	hypervisor.WindowsServiceDisablers.Register("hyperv", &Services{})
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
	return []string{
		"vmicheartbeat", "vmicshutdown",
		"vmicvss", "vmictimesync", "vmicrdv",
		"vmicguestinterface", "vmickvpexchange",
		"vmicvmsession", "storflt",
	}
}

func (s *Services) DisableServices(_ string, hive registry.Hive, ccs string) error {
	for _, svc := range s.ServiceNames() {
		svcPath := fmt.Sprintf("%s\\Services\\%s", ccs, svc)
		if hive.KeyExists(svcPath) {
			hive.SetDWORD(svcPath, "Start", 4)
			slog.Info("disabled service", "service", svc)
		}
	}

	timeProvider := ccs + `\Services\W32Time\TimeProviders\VMICTimeProvider`
	if hive.KeyExists(timeProvider) {
		hive.SetDWORD(timeProvider, "Enabled", 0)
		slog.Info("disabled Hyper-V time provider", "path", timeProvider)
	}
	return nil
}
