//go:build unix

package nutanix

import (
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

const uninstallKey = "Microsoft\\Windows\\CurrentVersion\\Uninstall\\Nutanix Guest Tools"

type Remove struct{}

func init() {
	hypervisor.WindowsRemoves.Register("nutanix", &Remove{})
}

func (r *Remove) Detect(guestRoot string, _, softwareHive registry.Hive) bool {
	for _, sub := range []string{"Program Files", "Program Files (x86)"} {
		if guestio.FileExists(filepath.Join(guestRoot, sub, "Nutanix")) {
			return true
		}
	}
	return softwareHive.KeyExists(uninstallKey)
}

func (r *Remove) Remove(guestRoot string, _, softwareHive registry.Hive) error {
	for _, sub := range []string{"Program Files", "Program Files (x86)"} {
		_ = guestio.FileRemoveAll(filepath.Join(guestRoot, sub, "Nutanix"))
	}
	softwareHive.DeleteKey(uninstallKey)
	return nil
}
