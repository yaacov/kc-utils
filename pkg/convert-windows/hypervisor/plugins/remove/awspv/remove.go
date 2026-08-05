//go:build linux

package awspv

import (
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor"
	"github.com/yaacov/kc-utils/pkg/guest"
)

const uninstallKey = `Microsoft\Windows\CurrentVersion\Uninstall\AWS PV Drivers`

type Remove struct{}

func init() {
	hypervisor.WindowsRemoves.Register("awspv", &Remove{})
}

func (r *Remove) Detect(guestRoot string, _, softwareHive registry.Hive) bool {
	if softwareHive.KeyExists(uninstallKey) {
		return true
	}
	driversDir := filepath.Join(guestRoot, "Windows", "System32", "drivers")
	entries, err := guest.FileReadDir(driversDir)
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

func (r *Remove) Remove(guestRoot string, _, softwareHive registry.Hive) error {
	softwareHive.DeleteKey(uninstallKey)

	driversDir := filepath.Join(guestRoot, "Windows", "System32", "drivers")
	entries, err := guest.FileReadDir(driversDir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		name := strings.ToLower(e.Name)
		if strings.HasPrefix(name, "xen") && strings.HasSuffix(name, ".sys") {
			path := filepath.Join(driversDir, e.Name)
			if rmErr := guest.FileRemove(path); rmErr != nil {
				slog.Warn("removing driver file", "path", path, "error", rmErr)
			}
		}
	}

	return nil
}
