//go:build unix

package drivers

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/convert-windows/driversource"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

// Copy copies VirtIO driver package files into the guest and returns copied driver names.
// Only files listed in each DriverFile.Files entry are staged. Boot-critical .sys files
// are also copied to system32\drivers for pre-PnP boot.
func Copy(mountRoot string, driverFiles []driversource.DriverFile) ([]string, error) {
	virtioDir := filepath.Join(mountRoot, "Windows", "Drivers", "VirtIO")
	sysDriversDir := filepath.Join(mountRoot, "Windows", "System32", "drivers")
	var copiedDriverNames []string
	if len(driverFiles) == 0 {
		return copiedDriverNames, nil
	}
	if mkErr := guestio.FileMkdirAll(virtioDir, 0o755); mkErr != nil {
		return nil, fmt.Errorf("creating VirtIO driver dir: %w", mkErr)
	}
	if mkErr := guestio.FileMkdirAll(sysDriversDir, 0o755); mkErr != nil {
		return nil, fmt.Errorf("creating system32 drivers dir: %w", mkErr)
	}
	for _, df := range driverFiles {
		if len(df.Files) == 0 {
			slog.Warn("skipping driver with empty Files list", "name", df.Name)
			continue
		}
		pkgRoot := df.SrcPath
		if pkgRoot == "" {
			pkgRoot = filepath.Dir(df.InfPath)
		}
		copiedAny := false
		for _, srcFile := range df.Files {
			rel := filepath.Base(srcFile)
			if pkgRoot != "" {
				if r, relErr := filepath.Rel(pkgRoot, srcFile); relErr == nil &&
					r != "." && !strings.HasPrefix(r, "..") {
					rel = r
				}
			}
			dstFile := filepath.Join(virtioDir, rel)
			if wrErr := guestio.FileUpload(srcFile, dstFile); wrErr != nil {
				slog.Warn("writing driver file failed", "path", dstFile, "error", wrErr)
				continue
			}
			copiedAny = true
			base := filepath.Base(srcFile)
			lower := strings.ToLower(base)
			if strings.HasSuffix(lower, ".sys") && BootCriticalDrivers[strings.TrimSuffix(lower, ".sys")] {
				sysDst := filepath.Join(sysDriversDir, base)
				if wrErr := guestio.FileUpload(srcFile, sysDst); wrErr != nil {
					slog.Warn("writing system32 driver failed", "path", sysDst, "error", wrErr)
				}
			}
		}
		if !copiedAny {
			slog.Warn("no files copied for driver", "name", df.Name)
			continue
		}
		copiedDriverNames = append(copiedDriverNames, df.Name)
		slog.Info("copied driver", "name", df.Name, "files", len(df.Files))
	}
	return copiedDriverNames, nil
}
