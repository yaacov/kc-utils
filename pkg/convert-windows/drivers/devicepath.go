package drivers

import (
	"log/slog"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/registry"
)

// Update appends the VirtIO driver path to DevicePath if not already present.
func Update(softwareHive registry.Hive) {
	devicePathKey := `Microsoft\Windows\CurrentVersion`
	virtioDevPath := `%SystemRoot%\Drivers\VirtIO`
	currentDevPath, devErr := softwareHive.GetString(devicePathKey, "DevicePath")
	if devErr != nil {
		slog.Warn("reading DevicePath failed, setting default", "error", devErr)
		currentDevPath = `%SystemRoot%\inf`
	}
	if !strings.Contains(currentDevPath, virtioDevPath) {
		newDevPath := currentDevPath + ";" + virtioDevPath
		softwareHive.SetExpandString(devicePathKey, "DevicePath", newDevPath)
		slog.Info("updated DevicePath", "path", newDevPath)
	}
}
