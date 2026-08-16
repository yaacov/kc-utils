//go:build unix

package nm

import (
	"bufio"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

type Plugin struct{}

func init() {
	nicnaming.Namers.Register("nm", &Plugin{})
}

func (p *Plugin) Detect(guestRoot string) bool {
	dir := filepath.Join(guestRoot, "etc", "NetworkManager", "system-connections")
	return guestio.FileIsDir(dir)
}

func (p *Plugin) ResolveNames(guestRoot string, entries []nicnaming.MacIPEntry) ([]nicnaming.NamingRule, error) {
	connDir := filepath.Join(guestRoot, "etc", "NetworkManager", "system-connections")
	files, err := guestio.FileReadDir(connDir)
	if err != nil {
		return nil, nil
	}

	var rules []nicnaming.NamingRule
	for _, entry := range entries {
		for _, f := range files {
			if f.IsDir {
				continue
			}
			path := filepath.Join(connDir, f.Name)
			if !nmFileHasIP(path, entry.IP) {
				continue
			}
			device := nmFileDevice(path)
			if device != "" {
				rules = append(rules, nicnaming.NamingRule{MAC: entry.MAC, Device: device})
				break
			}
		}
	}
	return rules, nil
}

func nmFileHasIP(path, ip string) bool {
	data, err := guestio.FileRead(path)
	if err != nil {
		return false
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "address") && strings.Contains(line, ip) {
			return true
		}
	}
	return false
}

func nmFileDevice(path string) string {
	data, err := guestio.FileRead(path)
	if err != nil {
		return ""
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "interface-name=") {
			return strings.TrimPrefix(line, "interface-name=")
		}
	}
	return ""
}
