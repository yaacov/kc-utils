//go:build linux

package parallels

import (
	"log/slog"
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor"
	"github.com/yaacov/kc-utils/pkg/guest"
)

const uninstallKey = `Microsoft\Windows\CurrentVersion\Uninstall\Parallels Tools`

type Remove struct{}

func init() {
	hypervisor.WindowsRemoves.Register("parallels", &Remove{})
}

func (r *Remove) Detect(guestRoot string, _ registry.Hive, softwareHive registry.Hive) bool {
	if guest.FileExists(filepath.Join(guestRoot, "Program Files", "Parallels", "Parallels Tools")) {
		return true
	}
	return softwareHive.KeyExists(uninstallKey)
}

func (r *Remove) Remove(_ string, _ registry.Hive, _ registry.Hive) error {
	slog.Warn("Parallels guest software removal not yet implemented, skipping")
	return nil
}
