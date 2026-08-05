//go:build linux

package dynamicscripts

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/firstboot"
	"github.com/yaacov/kc-utils/pkg/finalize/customize"
	"github.com/yaacov/kc-utils/pkg/guest"
)

var (
	linuxRegex   = regexp.MustCompile(`^([0-9]+)_linux_(run|firstboot)(([\w\-]*)\.sh)$`)
	windowsRegex = regexp.MustCompile(`^([0-9]+)_win_firstboot(([\w\-]*)\.ps1)$`)
)

type script struct {
	Priority int
	Action   string
	Path     string
	Name     string
}

type DynamicScripts struct{}

func init() {
	customize.Customizers.Register("dynamicscripts", &DynamicScripts{})
}

func (d *DynamicScripts) Apply(guestRoot string, options map[string]string) error {
	dir := options["scripts_dir"]
	if dir == "" {
		dir = "/mnt/dynamic_scripts"
	}
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}

	osType := options["os_type"]
	scripts, err := scanScripts(dir, osType)
	if err != nil {
		return err
	}
	slog.Info("scanned dynamic scripts", "dir", dir, "osType", osType, "count", len(scripts))
	for _, s := range scripts {
		slog.Info("applying dynamic script",
			"script", s.Name,
			"action", s.Action,
			"priority", s.Priority,
		)
		if err := applyScript(guestRoot, s); err != nil {
			slog.Warn("dynamic script failed", "script", s.Name, "error", err)
			continue
		}
		slog.Info("dynamic script complete", "script", s.Name, "action", s.Action)
	}
	return nil
}

func scanScripts(dir, osType string) ([]script, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var scripts []script
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		var re *regexp.Regexp
		switch osType {
		case "windows":
			re = windowsRegex
		default:
			re = linuxRegex
		}
		m := re.FindStringSubmatch(name)
		if m == nil {
			slog.Debug("skipping non-matching script", "name", name)
			continue
		}
		priority, _ := strconv.Atoi(m[1])
		action := m[2]
		if osType == "windows" {
			action = "win-firstboot"
		}
		scripts = append(scripts, script{
			Priority: priority,
			Action:   action,
			Path:     filepath.Join(dir, name),
			Name:     name,
		})
	}
	sort.Slice(scripts, func(i, j int) bool {
		return scripts[i].Priority < scripts[j].Priority
	})
	return scripts, nil
}

func applyScript(guestRoot string, s script) error {
	switch s.Action {
	case "run":
		return runLinuxScript(guestRoot, s.Path)
	case "firstboot":
		return installLinuxFirstboot(guestRoot, s)
	case "win-firstboot":
		return installWindowsFirstboot(guestRoot, s)
	default:
		return fmt.Errorf("unknown action %q", s.Action)
	}
}

func runLinuxScript(guestRoot, scriptPath string) error {
	guestScript := filepath.Join(guestRoot, "tmp", filepath.Base(scriptPath))
	if err := guest.FileMkdirAll(filepath.Dir(guestScript), 0o755); err != nil {
		return err
	}
	if err := guest.FileUpload(scriptPath, guestScript); err != nil {
		return err
	}
	if err := guest.FileChmod(guestScript, 0o755); err != nil {
		return err
	}
	_, err := guest.RunInGuest(guestRoot, []string{"/bin/bash", "/tmp/" + filepath.Base(scriptPath)})
	return err
}

func installLinuxFirstboot(guestRoot string, s script) error {
	destDir := filepath.Join(guestRoot, "usr", "local", "bin")
	if err := guest.FileMkdirAll(destDir, 0o755); err != nil {
		return err
	}
	destName := fmt.Sprintf("kc-dynamic-%s", s.Name)
	destPath := filepath.Join(destDir, destName)
	if err := guest.FileUpload(s.Path, destPath); err != nil {
		return err
	}
	if err := guest.FileChmod(destPath, 0o755); err != nil {
		return err
	}

	firstbootPath := filepath.Join(destDir, "kc-firstboot.sh")
	cmdLine := fmt.Sprintf(`run_with_retry "bash %s"`, destPath)
	if existing, err := guest.FileRead(firstbootPath); err == nil {
		content := strings.TrimSuffix(string(existing), "\n") + "\n" + cmdLine + "\n"
		return guest.FileWrite(firstbootPath, []byte(content), 0o755)
	}

	handler, ok := firstboot.Handlers.Get("systemd")
	if !ok {
		return fmt.Errorf("systemd firstboot handler not registered")
	}
	return handler.Install(guestRoot, []string{cmdLine})
}

func installWindowsFirstboot(guestRoot string, s script) error {
	destDir := filepath.Join(guestRoot, "Program Files", "Guestfs", "Firstboot", "scripts")
	if err := guest.FileMkdirAll(destDir, 0o755); err != nil {
		return err
	}
	return guest.FileUpload(s.Path, filepath.Join(destDir, s.Name))
}
