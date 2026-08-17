//go:build unix

package awspv

import (
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

const (
	uninstallKey    = `Microsoft\Windows\CurrentVersion\Uninstall\AWS PV Drivers`
	systemClassGUID = `{4d36e97d-e325-11ce-bfc1-08002be10318}`
	hdcClassGUID    = `{4d36e96a-e325-11ce-bfc1-08002be10318}`
)

type Remove struct{}

func init() {
	hypervisor.WindowsRemoves.Register("awspv", &Remove{})
}

func (r *Remove) Detect(guestRoot string, _, softwareHive registry.Hive) bool {
	if softwareHive.KeyExists(uninstallKey) {
		return true
	}
	driversDir := filepath.Join(guestRoot, "Windows", "System32", "drivers")
	entries, err := guestio.FileReadDir(driversDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		name := strings.ToLower(e.Name)
		if strings.HasPrefix(name, "xen") && strings.HasSuffix(name, ".sys") {
			return true
		}
	}
	return false
}

func (r *Remove) Remove(guestRoot string, systemHive, softwareHive registry.Hive) error {
	softwareHive.DeleteKey(uninstallKey)

	if systemHive != nil {
		ccs := hypervisor.CurrentControlSet(systemHive)
		// AWS PV installs XENFILT as an UpperFilter on System/HDC classes.
		// Deleting xenfilt.sys without clearing these entries leaves Windows
		// unable to attach the boot disk (INACCESSIBLE_BOOT_DEVICE).
		for _, guid := range []string{systemClassGUID, hdcClassGUID} {
			classPath := ccs + `\Control\Class\` + guid
			hypervisor.RemoveFilter(systemHive, classPath, "UpperFilters", "XENFILT")
		}
		hypervisor.DisableService(systemHive, ccs, "xenfilt")
	}

	driversDir := filepath.Join(guestRoot, "Windows", "System32", "drivers")
	entries, err := guestio.FileReadDir(driversDir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		name := strings.ToLower(e.Name)
		if strings.HasPrefix(name, "xen") && strings.HasSuffix(name, ".sys") {
			path := filepath.Join(driversDir, e.Name)
			if rmErr := guestio.FileRemove(path); rmErr != nil {
				slog.Warn("removing driver file", "path", path, "error", rmErr)
			}
		}
	}

	return nil
}
