//go:build linux

package firstboot

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-windows/driversource"
	"github.com/yaacov/kc-utils/pkg/convert-windows/version"
	"github.com/yaacov/kc-utils/pkg/guest"
)

// Config holds firstboot script generation parameters.
type Config struct {
	MountRoot   string
	Offline     bool
	DriverFiles []driversource.DriverFile
	StaticIPs   []types.StaticIP
	Options     types.PrepareOptions
	Version     version.VersionHandler
}

// WriteScript writes a PowerShell firstboot script with a priority-based filename.
func WriteScript(baseDir string, priority int, name string, content string) error {
	return writeScriptFile(baseDir, priority, name, content, ".ps1")
}

// WriteBatScript writes a batch firstboot script with a priority-based filename.
func WriteBatScript(baseDir string, priority int, name string, content string) error {
	return writeScriptFile(baseDir, priority, name, content, ".bat")
}

func writeScriptFile(baseDir string, priority int, name, content, ext string) error {
	scriptsDir := filepath.Join(baseDir, "scripts")
	if err := guest.FileMkdirAll(scriptsDir, 0o755); err != nil {
		return fmt.Errorf("creating scripts dir: %w", err)
	}
	filename := fmt.Sprintf("%04d-%s%s", priority, name, ext)
	scriptPath := filepath.Join(scriptsDir, filename)
	return guest.FileWrite(scriptPath, []byte(content), 0o644)
}

// Configure generates firstboot scripts, launcher, and RunOnce registry entry.
func Configure(cfg *Config, softwareHive registry.Hive) error {
	firstbootDir := filepath.Join(cfg.MountRoot, "Program Files", "Guestfs", "Firstboot")
	if mkErr := guest.FileMkdirAll(filepath.Join(firstbootDir, "scripts"), 0o755); mkErr != nil {
		return fmt.Errorf("creating firstboot dir: %w", mkErr)
	}

	ver := cfg.Version
	if ver == nil {
		ver = version.Classify(nil)
	}

	staticIPs := cfg.StaticIPs
	if len(staticIPs) == 0 {
		staticIPs = cfg.Options.StaticIPs
	}

	contribCfg := &ContributorConfig{
		MountRoot:   cfg.MountRoot,
		Offline:     cfg.Offline,
		DriverFiles: cfg.DriverFiles,
		StaticIPs:   staticIPs,
		Options:     cfg.Options,
		Version:     ver,
	}

	type sortedContrib struct {
		priority int
		contrib  Contributor
	}
	var active []sortedContrib
	for _, c := range Contributors.All() {
		if c.ShouldRun(contribCfg) {
			active = append(active, sortedContrib{priority: c.Priority(), contrib: c})
		}
	}
	sort.Slice(active, func(i, j int) bool {
		return active[i].priority < active[j].priority
	})

	slog.Info("running firstboot contributors", "count", len(active), "version", ver.Name())
	for _, sc := range active {
		name := sc.contrib.Name()
		slog.Info("running firstboot contributor", "name", name, "priority", sc.priority)
		content, err := sc.contrib.Generate(contribCfg)
		if err != nil {
			slog.Warn("firstboot contributor failed", "name", name, "error", err)
			continue
		}
		if content == "" {
			slog.Debug("firstboot contributor produced empty script", "name", name)
			continue
		}
		var writeErr error
		if sc.contrib.UsesBatch(contribCfg) {
			writeErr = WriteBatScript(firstbootDir, sc.contrib.Priority(), name, content)
		} else {
			writeErr = WriteScript(firstbootDir, sc.contrib.Priority(), name, content)
		}
		if writeErr != nil {
			slog.Warn("writing firstboot script failed", "name", name, "error", writeErr)
			continue
		}
		slog.Info("wrote firstboot script", "name", name, "priority", sc.priority)
	}

	batContent := launcherScript(ver.FirstbootLauncher())
	batPath := filepath.Join(firstbootDir, "firstboot.bat")
	if batErr := guest.FileWrite(batPath, []byte(batContent), 0o644); batErr != nil {
		slog.Warn("writing firstboot.bat failed", "error", batErr)
	}

	runOncePath := `Microsoft\Windows\CurrentVersion\RunOnce`
	softwareHive.CreateKey(runOncePath)
	softwareHive.SetString(runOncePath, "kcfirstboot",
		`C:\Program Files\Guestfs\Firstboot\firstboot.bat`)

	return nil
}

func launcherScript(kind version.LauncherKind) string {
	switch kind {
	case version.LauncherPSV1:
		return psV1Launcher()
	case version.LauncherBatOnly:
		return batOnlyLauncher()
	default:
		return modernLauncher()
	}
}

func modernLauncher() string {
	return "@echo off\r\n" +
		"setlocal enabledelayedexpansion\r\n" +
		"cd /d \"%~dp0\"\r\n" +
		"for /f \"delims=\" %%s in ('dir /b /o:n \"scripts\\*.bat\" 2^>nul') do (\r\n" +
		"    echo Running %%s\r\n" +
		"    call \"%~dp0scripts\\%%s\"\r\n" +
		")\r\n" +
		"for /f \"delims=\" %%s in ('dir /b /o:n \"scripts\\*.ps1\" 2^>nul') do (\r\n" +
		"    echo Running %%s\r\n" +
		"    powershell.exe -ExecutionPolicy Bypass -File \"%~dp0scripts\\%%s\"\r\n" +
		")\r\n" +
		cleanupFooter()
}

func psV1Launcher() string {
	return "@echo off\r\n" +
		"setlocal enabledelayedexpansion\r\n" +
		"cd /d \"%~dp0\"\r\n" +
		"reg add \"HKLM\\SOFTWARE\\Microsoft\\PowerShell\\1\\ShellIds\\Microsoft.PowerShell\" /v ExecutionPolicy /t REG_SZ /d RemoteSigned /f >nul 2>&1\r\n" +
		"for /f \"delims=\" %%s in ('dir /b /o:n \"scripts\\*.bat\" 2^>nul') do (\r\n" +
		"    echo Running %%s\r\n" +
		"    call \"%~dp0scripts\\%%s\"\r\n" +
		")\r\n" +
		"for /f \"delims=\" %%s in ('dir /b /o:n \"scripts\\*.ps1\" 2^>nul') do (\r\n" +
		"    echo Running %%s\r\n" +
		"    powershell.exe -File \"%~dp0scripts\\%%s\"\r\n" +
		")\r\n" +
		cleanupFooter()
}

func batOnlyLauncher() string {
	return "@echo off\r\n" +
		"setlocal enabledelayedexpansion\r\n" +
		"cd /d \"%~dp0\"\r\n" +
		"for /f \"delims=\" %%s in ('dir /b /o:n \"scripts\\*.bat\" 2^>nul') do (\r\n" +
		"    echo Running %%s\r\n" +
		"    call \"%~dp0scripts\\%%s\"\r\n" +
		")\r\n" +
		cleanupFooter()
}

func cleanupFooter() string {
	return "cd /d \"%TEMP%\"\r\n" +
		"rmdir /s /q \"C:\\Program Files\\Guestfs\\Firstboot\" 2>nul\r\n" +
		"rmdir \"C:\\Program Files\\Guestfs\" 2>nul\r\n"
}
