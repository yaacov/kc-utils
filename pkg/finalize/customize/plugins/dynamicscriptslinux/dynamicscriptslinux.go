//go:build unix

package dynamicscriptslinux

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

	"github.com/yaacov/kc-utils/pkg/common/firstboot"
	"github.com/yaacov/kc-utils/pkg/finalize/customize"
	"github.com/yaacov/kc-utils/pkg/guest"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

var scriptRegex = regexp.MustCompile(`^([0-9]+)_linux_(run|firstboot)(([\w\-]*)\.sh)$`)

type script struct {
	Priority int
	Action   string
	Path     string
	Name     string
}

type DynamicScriptsLinux struct{}

func init() {
	customize.Customizers.Register("dynamicscriptslinux", &DynamicScriptsLinux{})
}

func (d *DynamicScriptsLinux) Apply(guestRoot string, options map[string]string) error {
	if options["os_type"] != "linux" {
		return nil
	}

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

	scripts, err := scanScripts(dir)
	if err != nil {
		return err
	}
	slog.Info("scanned dynamic scripts", "dir", dir, "osType", "linux", "count", len(scripts))
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

func scanScripts(dir string) ([]script, error) {
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
		m := scriptRegex.FindStringSubmatch(name)
		if m == nil {
			slog.Debug("skipping non-matching script", "name", name)
			continue
		}
		priority, _ := strconv.Atoi(m[1])
		scripts = append(scripts, script{
			Priority: priority,
			Action:   m[2],
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
		return runScript(guestRoot, s.Path)
	case "firstboot":
		return installFirstboot(guestRoot, s)
	default:
		return fmt.Errorf("unknown action %q", s.Action)
	}
}

func runScript(guestRoot, scriptPath string) error {
	guestScript := filepath.Join(guestRoot, "tmp", filepath.Base(scriptPath))
	if err := guestio.FileMkdirAll(filepath.Dir(guestScript), 0o755); err != nil {
		return err
	}
	if err := guestio.FileUpload(scriptPath, guestScript); err != nil {
		return err
	}
	if err := guestio.FileChmod(guestScript, 0o755); err != nil {
		return err
	}
	_, err := guest.RunInGuest(guestRoot, []string{"/bin/bash", "/tmp/" + filepath.Base(scriptPath)})
	return err
}

func installFirstboot(guestRoot string, s script) error {
	destDir := filepath.Join(guestRoot, "usr", "local", "bin")
	if err := guestio.FileMkdirAll(destDir, 0o755); err != nil {
		return err
	}
	destName := fmt.Sprintf("kc-dynamic-%s", s.Name)
	destPath := filepath.Join(destDir, destName)
	if err := guestio.FileUpload(s.Path, destPath); err != nil {
		return err
	}
	if err := guestio.FileChmod(destPath, 0o755); err != nil {
		return err
	}

	handler, ok := firstboot.Handlers.Get("systemd")
	if !ok {
		return fmt.Errorf("systemd firstboot handler not registered")
	}
	// The command runs inside the booted guest, so reference the guest-absolute
	// path, not the host mount path (destPath). Pass the bare command: the
	// handler adds run_with_retry wrapping and appends it before the script's
	// self-cleanup tail (append-safe across multiple firstboot scripts).
	cmdLine := fmt.Sprintf("bash /usr/local/bin/%s", destName)
	return handler.Install(guestRoot, []string{cmdLine})
}
