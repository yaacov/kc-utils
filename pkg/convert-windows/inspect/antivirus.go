package inspect

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/registry"
)

// DetectAntivirus scans installed software for antivirus products.
// It logs warnings and returns messages suitable for ConverterOutput.Warnings.
func DetectAntivirus(softwareHive registry.Hive) []string {
	var warnings []string
	uninstallPath := `Microsoft\Windows\CurrentVersion\Uninstall`
	keys, enumErr := softwareHive.EnumKeys(uninstallPath)
	if enumErr != nil {
		slog.Warn("enumerating uninstall keys failed", "error", enumErr)
		return warnings
	}
	for _, key := range keys {
		keyPath := uninstallPath + `\` + key
		displayName, nameErr := softwareHive.GetString(keyPath, "DisplayName")
		if nameErr != nil {
			continue
		}
		lower := strings.ToLower(displayName)
		if strings.Contains(lower, "antivirus") ||
			strings.Contains(lower, "anti-virus") ||
			strings.Contains(lower, "endpoint protection") ||
			strings.Contains(lower, "security center") {
			msg := fmt.Sprintf("antivirus detected: %s; this may interfere with virtio driver installation (INACCESSIBLE_BOOT_DEVICE)", displayName)
			slog.Warn(msg)
			warnings = append(warnings, msg)
		}
	}
	return warnings
}
