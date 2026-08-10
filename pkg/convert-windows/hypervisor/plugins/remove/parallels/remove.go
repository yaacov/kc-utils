//go:build linux

package parallels

import (
	"log/slog"
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor"
	"github.com/yaacov/kc-utils/pkg/guest"
)

const (
	uninstallKey  = `Microsoft\Windows\CurrentVersion\Uninstall\Parallels Tools`
	diskClassGUID = `{4d36e967-e325-11ce-bfc1-08002be10318}`
)

type Remove struct{}

func init() {
	hypervisor.WindowsRemoves.Register("parallels", &Remove{})
}

func (r *Remove) Detect(guestRoot string, _ registry.Hive, softwareHive registry.Hive) bool {
	if guest.FileExists(filepath.Join(guestRoot, "Program Files", "Parallels", "Parallels Tools")) ||
		guest.FileExists(filepath.Join(guestRoot, "Program Files (x86)", "Parallels", "Parallels Tools")) {
		return true
	}
	return softwareHive.KeyExists(uninstallKey)
}

func (r *Remove) Remove(guestRoot string, systemHive, softwareHive registry.Hive) error {
	ccs := hypervisor.CurrentControlSet(systemHive)
	classPath := ccs + `\Control\Class\` + diskClassGUID
	hypervisor.RemoveFilter(systemHive, classPath, "LowerFilters", "prl_strg")

	for _, svc := range []string{
		"prl_strg", "prl_boot", "prl_scsi", "prl_eth5", "Parallels Tools Service",
	} {
		hypervisor.DisableService(systemHive, ccs, svc)
	}

	for _, sub := range []string{"Program Files", "Program Files (x86)"} {
		p := filepath.Join(guestRoot, sub, "Parallels", "Parallels Tools")
		if guest.FileExists(p) {
			_ = guest.FileRemoveAll(p)
			slog.Info("removed Parallels Tools directory", "path", p)
		}
	}

	if softwareHive.KeyExists(uninstallKey) {
		softwareHive.DeleteKey(uninstallKey)
	}

	slog.Info("Parallels cleanup complete")
	return nil
}
