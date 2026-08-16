//go:build unix

package virtualbox

import (
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor"
	"github.com/yaacov/kc-utils/pkg/guest"
)

const uninstallKey = `Microsoft\Windows\CurrentVersion\Uninstall\Oracle VM VirtualBox Guest Additions`

type Remove struct{}

func init() {
	hypervisor.WindowsRemoves.Register("virtualbox", &Remove{})
}

func (r *Remove) Detect(guestRoot string, _, softwareHive registry.Hive) bool {
	gaDir := filepath.Join(guestRoot, "Program Files", "Oracle", "VirtualBox Guest Additions")
	return guest.FileExists(gaDir) || softwareHive.KeyExists(uninstallKey)
}

func (r *Remove) Remove(guestRoot string, _, softwareHive registry.Hive) error {
	gaDir := filepath.Join(guestRoot, "Program Files", "Oracle", "VirtualBox Guest Additions")
	_ = guest.FileRemoveAll(gaDir)

	softwareHive.DeleteKey(uninstallKey)

	driversDir := filepath.Join(guestRoot, "Windows", "System32", "drivers")
	entries, err := guest.FileReadDir(driversDir)
	if err == nil {
		for _, e := range entries {
			name := strings.ToLower(e.Name)
			if strings.HasPrefix(name, "vbox") && strings.HasSuffix(name, ".sys") {
				path := filepath.Join(driversDir, e.Name)
				if rmErr := guest.FileRemove(path); rmErr != nil {
					slog.Warn("removing VBox driver", "path", path, "error", rmErr)
				}
			}
		}
	}
	return nil
}
