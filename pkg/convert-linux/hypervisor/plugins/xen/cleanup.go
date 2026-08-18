//go:build unix

package xen

import (
	"bufio"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type Cleanup struct{}

func init() {
	hypervisor.LinuxCleanups.Register("xen", &Cleanup{})
}

var xenModules = map[string]bool{
	"xennet": true, "xen-vnif": true, "xenblk": true, "xen-vbd": true,
}

func (c *Cleanup) Detect(guestRoot string) bool {
	path := filepath.Join(guestRoot, "etc", "sysconfig", "kernel")
	data, err := guest.FileRead(path)
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(data))
	for mod := range xenModules {
		if strings.Contains(lower, mod) {
			return true
		}
	}
	return false
}

func (c *Cleanup) Cleanup(guestRoot string) error {
	path := filepath.Join(guestRoot, "etc", "sysconfig", "kernel")
	data, err := guest.FileRead(path)
	if err != nil {
		return nil
	}

	var out strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	changed := false
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "INITRD_MODULES=") || strings.HasPrefix(trimmed, "DOMU_INITRD_MODULES=") {
			eq := strings.Index(line, "=")
			if eq < 0 {
				out.WriteString(line)
				out.WriteByte('\n')
				continue
			}
			key := line[:eq+1]
			val := strings.Trim(line[eq+1:], `"'`)
			fields := strings.Fields(val)
			var kept []string
			for _, f := range fields {
				if xenModules[f] {
					changed = true
					continue
				}
				kept = append(kept, f)
			}
			out.WriteString(key)
			out.WriteString(`"`)
			out.WriteString(strings.Join(kept, " "))
			out.WriteString(`"`)
			out.WriteByte('\n')
			continue
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	if changed {
		if err := guest.FileWrite(path, []byte(out.String()), 0o644); err != nil {
			slog.Warn("writing cleaned xen kernel config failed", "path", path, "error", err)
		}
	}
	return nil
}
