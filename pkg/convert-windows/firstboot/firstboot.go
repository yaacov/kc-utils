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
	"github.com/yaacov/kc-utils/pkg/guest"
)

// Config holds firstboot script generation parameters.
type Config struct {
	MountRoot   string
	Offline     bool
	DriverFiles []driversource.DriverFile
	StaticIPs   []types.StaticIP
	Options     types.PrepareOptions
}

// WriteScript writes a PowerShell firstboot script with a priority-based filename.
func WriteScript(baseDir string, priority int, name string, content string) error {
	scriptsDir := filepath.Join(baseDir, "scripts")
	if err := guest.FileMkdirAll(scriptsDir, 0o755); err != nil {
		return fmt.Errorf("creating scripts dir: %w", err)
	}
	filename := fmt.Sprintf("%04d-%s.ps1", priority, name)
	scriptPath := filepath.Join(scriptsDir, filename)
	return guest.FileWrite(scriptPath, []byte(content), 0o644)
}

// Configure generates firstboot scripts, launcher, and RunOnce registry entry.
func Configure(cfg *Config, softwareHive registry.Hive) error {
	firstbootDir := filepath.Join(cfg.MountRoot, "Program Files", "Guestfs", "Firstboot")
	if mkErr := guest.FileMkdirAll(filepath.Join(firstbootDir, "scripts"), 0o755); mkErr != nil {
		return fmt.Errorf("creating firstboot dir: %w", mkErr)
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

	slog.Info("running firstboot contributors", "count", len(active))
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
		if fbErr := WriteScript(firstbootDir, sc.contrib.Priority(), name, content); fbErr != nil {
			slog.Warn("writing firstboot script failed", "name", name, "error", fbErr)
			continue
		}
		slog.Info("wrote firstboot script", "name", name, "priority", sc.priority)
	}

	batContent := "@echo off\r\n" +
		"setlocal enabledelayedexpansion\r\n" +
		"cd /d \"%~dp0\"\r\n" +
		"for /f \"delims=\" %%s in ('dir /b /o:n \"scripts\\*.ps1\" 2^>nul') do (\r\n" +
		"    echo Running %%s\r\n" +
		"    powershell.exe -ExecutionPolicy Bypass -File \"%~dp0scripts\\%%s\"\r\n" +
		")\r\n" +
		"cd /d \"%TEMP%\"\r\n" +
		"rmdir /s /q \"C:\\Program Files\\Guestfs\\Firstboot\" 2>nul\r\n" +
		"rmdir \"C:\\Program Files\\Guestfs\" 2>nul\r\n"
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
