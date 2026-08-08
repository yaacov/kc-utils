//go:build linux

package citrix

import (
	"log/slog"
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor"
	"github.com/yaacov/kc-utils/pkg/guest"
)

const uninstallKey = `Microsoft\Windows\CurrentVersion\Uninstall\Citrix XenTools`

type Remove struct{}

func init() {
	hypervisor.WindowsRemoves.Register("citrix", &Remove{})
}

func (r *Remove) Detect(guestRoot string, _ registry.Hive, softwareHive registry.Hive) bool {
	if guest.FileExists(filepath.Join(guestRoot, "Program Files", "Citrix", "XenTools")) {
		return true
	}
	return softwareHive.KeyExists(uninstallKey)
}

func (r *Remove) Remove(_ string, _ registry.Hive, _ registry.Hive) error {
	slog.Warn("Citrix/XenServer guest software removal not yet implemented, skipping")
	return nil
}
