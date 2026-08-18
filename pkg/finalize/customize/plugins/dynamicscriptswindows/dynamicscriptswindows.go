//go:build unix

package dynamicscriptswindows

import (
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

	"github.com/yaacov/kc-utils/pkg/finalize/customize"
	"github.com/yaacov/kc-utils/pkg/guest"
)

var scriptRegex = regexp.MustCompile(`^([0-9]+)_win_firstboot(([\w\-]*)\.ps1)$`)

type script struct {
	Priority int
	Path     string
	Name     string
}

type DynamicScriptsWindows struct{}

func init() {
	customize.Customizers.Register("dynamicscriptswindows", &DynamicScriptsWindows{})
}

func (d *DynamicScriptsWindows) Apply(guestRoot string, options map[string]string) error {
	if options["os_type"] != "windows" {
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
	slog.Info("scanned dynamic scripts", "dir", dir, "osType", "windows", "count", len(scripts))
	for _, s := range scripts {
		slog.Info("applying dynamic script",
			"script", s.Name,
			"action", "firstboot",
			"priority", s.Priority,
		)
		if err := installFirstboot(guestRoot, s); err != nil {
			slog.Warn("dynamic script failed", "script", s.Name, "error", err)
			continue
		}
		slog.Info("dynamic script complete", "script", s.Name, "action", "firstboot")
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
			Path:     filepath.Join(dir, name),
			Name:     name,
		})
	}
	sort.Slice(scripts, func(i, j int) bool {
		return scripts[i].Priority < scripts[j].Priority
	})
	return scripts, nil
}

func installFirstboot(guestRoot string, s script) error {
	destDir := filepath.Join(guestRoot, "Program Files", "Guestfs", "Firstboot", "scripts")
	if err := guest.FileMkdirAll(destDir, 0o755); err != nil {
		return err
	}
	return guest.FileUpload(s.Path, filepath.Join(destDir, s.Name))
}
