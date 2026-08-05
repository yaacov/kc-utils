package directory

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/convert-windows/driversource"
)

const (
	DefaultBasePath      = "/usr/share/virtio-win/drivers/by-os"
	DefaultGuestAgentDir = "/usr/share/virtio-win/guest-agent"
)

type DirectorySource struct {
	BasePath      string
	GuestAgentDir string
}

func init() {
	driversource.Sources.Register("directory", &DirectorySource{})
}

func (d *DirectorySource) basePath() string {
	if d.BasePath != "" {
		return d.BasePath
	}
	return DefaultBasePath
}

func (d *DirectorySource) guestAgentDir() string {
	if d.GuestAgentDir != "" {
		return d.GuestAgentDir
	}
	return DefaultGuestAgentDir
}

func (d *DirectorySource) Available() bool {
	_, err := os.Stat(d.basePath())
	return err == nil
}

func (d *DirectorySource) FindDrivers(arch, osVersion string, osPrefs, osFallbacks []string) ([]driversource.DriverFile, error) {
	var drivers []driversource.DriverFile
	var lastErr error

	for _, archName := range driversource.ArchSearchNames(arch) {
		archDir := filepath.Join(d.basePath(), archName)
		if st, err := os.Stat(archDir); err != nil || !st.IsDir() {
			if lastErr == nil {
				lastErr = err
			}
			continue
		}

		osDir, err := driversource.FindBestOSDirWithPrefs(archDir, osVersion, osPrefs, osFallbacks)
		if err != nil {
			// Arch dir exists but no matching OS tree — do not fall through to
			// alternate arch names (e.g. x86_64) which would mask the real error.
			return nil, fmt.Errorf("no virtio-win drivers for arch=%s os=%s under %s: %w",
				arch, osVersion, d.basePath(), err)
		}

		found, err := collectInfDrivers(osDir, arch)
		if err != nil {
			return nil, fmt.Errorf("no virtio-win drivers for arch=%s os=%s under %s: %w",
				arch, osVersion, d.basePath(), err)
		}
		drivers = append(drivers, found...)
		break
	}

	if len(drivers) == 0 {
		if lastErr != nil {
			return nil, fmt.Errorf("no virtio-win drivers for arch=%s os=%s under %s: %w",
				arch, osVersion, d.basePath(), lastErr)
		}
		return nil, fmt.Errorf("no virtio-win drivers for arch=%s os=%s under %s", arch, osVersion, d.basePath())
	}

	gaDrivers, err := collectGuestAgentMSIs(d.guestAgentDir(), arch)
	if err != nil {
		return nil, err
	}
	drivers = append(drivers, gaDrivers...)

	slog.Info("found drivers", "count", len(drivers), "source", "directory", "arch", arch, "osVersion", osVersion)
	return drivers, nil
}

func collectInfDrivers(osDir, arch string) ([]driversource.DriverFile, error) {
	entries, err := os.ReadDir(osDir)
	if err != nil {
		return nil, fmt.Errorf("reading driver dir %s: %w", osDir, err)
	}

	var drivers []driversource.DriverFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".inf") {
			continue
		}
		drivers = append(drivers, driversource.DriverFile{
			Name:    strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())),
			SrcPath: osDir,
			InfPath: filepath.Join(osDir, entry.Name()),
			Arch:    arch,
		})
	}
	if len(drivers) == 0 {
		return nil, fmt.Errorf("no .inf drivers in %s", osDir)
	}
	return drivers, nil
}

func collectGuestAgentMSIs(dir, arch string) ([]driversource.DriverFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading guest-agent dir %s: %w", dir, err)
	}

	normArch := driversource.NormalizeArch(arch)
	var drivers []driversource.DriverFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if !strings.HasSuffix(name, ".msi") || !strings.Contains(name, "qemu-ga") {
			continue
		}
		if !msiMatchesArch(name, normArch) {
			continue
		}
		drivers = append(drivers, driversource.DriverFile{
			Name:    "qemu-ga",
			SrcPath: dir,
			InfPath: filepath.Join(dir, entry.Name()),
			Arch:    arch,
		})
	}
	return drivers, nil
}

func msiMatchesArch(msiName, normArch string) bool {
	switch normArch {
	case "amd64":
		return strings.Contains(msiName, "x64") ||
			strings.Contains(msiName, "x86_64") ||
			strings.Contains(msiName, "amd64") ||
			(!strings.Contains(msiName, "x86") && !strings.Contains(msiName, "i386"))
	case "x86":
		return strings.Contains(msiName, "x86") || strings.Contains(msiName, "i386")
	default:
		return true
	}
}
