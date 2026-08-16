//go:build unix

package citrix

import (
	"log/slog"
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

const (
	uninstallKey    = `Microsoft\Windows\CurrentVersion\Uninstall\Citrix XenTools`
	systemClassGUID = `{4d36e97d-e325-11ce-bfc1-08002be10318}`
	hdcClassGUID    = `{4d36e96a-e325-11ce-bfc1-08002be10318}`
)

type Remove struct{}

func init() {
	hypervisor.WindowsRemoves.Register("citrix", &Remove{})
}

func (r *Remove) Detect(guestRoot string, _ registry.Hive, softwareHive registry.Hive) bool {
	if guestio.FileExists(filepath.Join(guestRoot, "Program Files", "Citrix", "XenTools")) {
		return true
	}
	return softwareHive.KeyExists(uninstallKey)
}

func (r *Remove) Remove(guestRoot string, systemHive, softwareHive registry.Hive) error {
	ccs := hypervisor.CurrentControlSet(systemHive)
	for _, guid := range []string{systemClassGUID, hdcClassGUID} {
		classPath := ccs + `\Control\Class\` + guid
		hypervisor.RemoveFilter(systemHive, classPath, "UpperFilters", "XENFILT")
	}

	for _, svc := range []string{"XenSvc", "xenagent", "xenbus_monitor", "xenlite"} {
		hypervisor.DisableService(systemHive, ccs, svc)
	}

	toolsDir := filepath.Join(guestRoot, "Program Files", "Citrix", "XenTools")
	if guestio.FileExists(toolsDir) {
		_ = guestio.FileRemoveAll(toolsDir)
		slog.Info("removed Citrix XenTools directory", "path", toolsDir)
	}

	if softwareHive.KeyExists(uninstallKey) {
		softwareHive.DeleteKey(uninstallKey)
	}

	slog.Info("Citrix cleanup complete")
	return nil
}
