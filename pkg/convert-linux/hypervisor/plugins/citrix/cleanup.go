//go:build linux

package citrix

import (
	"bufio"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor"
	"github.com/yaacov/kc-utils/pkg/convert-linux/systemd"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type Cleanup struct{}

func init() {
	hypervisor.LinuxCleanups.Register("citrix", &Cleanup{})
}

func (c *Cleanup) Detect(guestRoot string) bool {
	indicators := []string{
		filepath.Join(guestRoot, "etc", "xensource-inventory"),
		filepath.Join(guestRoot, "usr", "sbin", "xe-daemon"),
	}
	for _, p := range indicators {
		if guest.FileExists(p) {
			return true
		}
	}
	return false
}

func (c *Cleanup) Cleanup(guestRoot string) error {
	systemd.DisableSystemdUnit(guestRoot, "xe-daemon.service")
	systemd.DisableSystemdUnit(guestRoot, "xapi.service")
	systemd.DisableSystemdUnit(guestRoot, "xe-linux-distribution.service")

	systemd.RemovePaths(
		filepath.Join(guestRoot, "etc", "xensource-inventory"),
		filepath.Join(guestRoot, "usr", "sbin", "xe-daemon"),
	)

	restoreInittabGettys(guestRoot)
	return nil
}

var inittabGettyComment = regexp.MustCompile(`^([1-6]):([2-5]+):respawn:(.*getty.*)`)

func restoreInittabGettys(guestRoot string) {
	path := filepath.Join(guestRoot, "etc", "inittab")
	data, err := guest.FileRead(path)
	if err != nil {
		return
	}

	var lines []string
	changed := false
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			body := strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
			if m := inittabGettyComment.FindStringSubmatch(body); m != nil {
				lines = append(lines, body)
				changed = true
				continue
			}
		}
		lines = append(lines, line)
	}
	if changed {
		if err := guest.FileWrite(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
			slog.Warn("writing cleaned citrix inittab failed", "path", path, "error", err)
		}
	}
}
