//go:build linux

package drivers

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/convert-windows/driversource"
	"github.com/yaacov/kc-utils/pkg/guest"
)

// Copy copies VirtIO driver files into the guest and returns copied driver names.
// Boot-critical .sys files are also copied to system32\drivers for pre-PnP boot.
// qemu-ga MSI packages are staged into the VirtIO directory for firstboot install.
func Copy(mountRoot string, driverFiles []driversource.DriverFile) ([]string, error) {
	virtioDir := filepath.Join(mountRoot, "Windows", "Drivers", "VirtIO")
	sysDriversDir := filepath.Join(mountRoot, "Windows", "System32", "drivers")
	var copiedDriverNames []string
	if len(driverFiles) == 0 {
		return copiedDriverNames, nil
	}
	if mkErr := guest.FileMkdirAll(virtioDir, 0o755); mkErr != nil {
		return nil, fmt.Errorf("creating VirtIO driver dir: %w", mkErr)
	}
	if mkErr := guest.FileMkdirAll(sysDriversDir, 0o755); mkErr != nil {
		return nil, fmt.Errorf("creating system32 drivers dir: %w", mkErr)
	}
	for _, df := range driverFiles {
		entries, readErr := os.ReadDir(df.SrcPath)
		if readErr != nil {
			slog.Warn("reading driver dir failed", "path", df.SrcPath, "error", readErr)
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			fname := entry.Name()
			lower := strings.ToLower(fname)
			if !strings.HasSuffix(lower, ".sys") &&
				!strings.HasSuffix(lower, ".inf") &&
				!strings.HasSuffix(lower, ".cat") &&
				!strings.HasSuffix(lower, ".msi") {
				continue
			}
			srcFile := filepath.Join(df.SrcPath, fname)
			dstFile := filepath.Join(virtioDir, fname)
			if wrErr := guest.FileUpload(srcFile, dstFile); wrErr != nil {
				slog.Warn("writing driver file failed", "path", dstFile, "error", wrErr)
				continue
			}
			// Boot storage drivers must also live under system32\drivers.
			if strings.HasSuffix(lower, ".sys") && bootCriticalDrivers[strings.TrimSuffix(lower, ".sys")] {
				sysDst := filepath.Join(sysDriversDir, fname)
				if wrErr := guest.FileUpload(srcFile, sysDst); wrErr != nil {
					slog.Warn("writing system32 driver failed", "path", sysDst, "error", wrErr)
				}
			}
		}
		copiedDriverNames = append(copiedDriverNames, df.Name)
		slog.Info("copied driver", "name", df.Name)
	}
	return copiedDriverNames, nil
}

var bootCriticalDrivers = map[string]bool{
	"viostor": true,
	"vioscsi": true,
}
