//go:build unix

package wicked

import (
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type Plugin struct{}

func init() {
	nicnaming.Namers.Register("wicked", &Plugin{})
}

func (p *Plugin) Detect(guestRoot string) bool {
	dir := filepath.Join(guestRoot, "var", "lib", "wicked")
	return guest.FileIsDir(dir)
}

func (p *Plugin) ResolveNames(guestRoot string, entries []nicnaming.MacIPEntry) ([]nicnaming.NamingRule, error) {
	wickedDir := filepath.Join(guestRoot, "var", "lib", "wicked")
	files, err := guest.FileReadDir(wickedDir)
	if err != nil {
		return nil, nil
	}

	var rules []nicnaming.NamingRule
	for _, entry := range entries {
		device := findDeviceByIP(wickedDir, files, entry.IP)
		if device != "" {
			rules = append(rules, nicnaming.NamingRule{MAC: entry.MAC, Device: device})
		}
	}
	return rules, nil
}

// findDeviceByIP searches wicked lease XML files for an <address> tag matching
// the given IP. Wicked lease filenames follow the pattern:
// lease-<interface>-dhcp-ipv4.xml
func findDeviceByIP(dir string, files []guest.DirEntry, ip string) string {
	searchTag := "<address>" + ip + "</address>"
	for _, f := range files {
		if f.IsDir {
			continue
		}
		path := filepath.Join(dir, f.Name)
		data, err := guest.FileRead(path)
		if err != nil {
			continue
		}
		if !strings.Contains(string(data), searchTag) {
			continue
		}
		// Extract interface name from filename: lease-<iface>-dhcp-...
		parts := strings.SplitN(f.Name, "-", 3)
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return ""
}
