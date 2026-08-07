//go:build linux

package ifcfg

import (
	"bufio"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type Plugin struct{}

func init() {
	nicnaming.Namers.Register("ifcfg", &Plugin{})
}

func (p *Plugin) Detect(guestRoot string) bool {
	for _, dir := range candidateDirs(guestRoot) {
		if guest.FileIsDir(dir) {
			return true
		}
	}
	return false
}

func (p *Plugin) ResolveNames(guestRoot string, entries []nicnaming.MacIPEntry) ([]nicnaming.NamingRule, error) {
	var rules []nicnaming.NamingRule

	for _, entry := range entries {
		for _, dir := range candidateDirs(guestRoot) {
			device := findDeviceByIP(dir, entry.IP, entry.MAC)
			if device != "" && device != "lo" {
				rules = append(rules, nicnaming.NamingRule{MAC: entry.MAC, Device: device})
				break
			}
		}
	}
	return rules, nil
}

func candidateDirs(guestRoot string) []string {
	return []string{
		filepath.Join(guestRoot, "etc", "sysconfig", "network-scripts"),
		filepath.Join(guestRoot, "etc", "sysconfig", "network"),
	}
}

func findDeviceByIP(scriptsDir, ip, mac string) string {
	entries, err := guest.FileReadDir(scriptsDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !strings.HasPrefix(e.Name, "ifcfg-") {
			continue
		}
		path := filepath.Join(scriptsDir, e.Name)
		if !fileContainsIP(path, ip) {
			continue
		}
		device := extractDevice(path, mac)
		if device != "" {
			return device
		}
		return strings.TrimPrefix(e.Name, "ifcfg-")
	}
	return ""
}

func fileContainsIP(path, ip string) bool {
	data, err := guest.FileRead(path)
	if err != nil {
		return false
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "IPADDR") && strings.Contains(line, ip) {
			return true
		}
	}
	return false
}

func extractDevice(path, _ string) string {
	data, err := guest.FileRead(path)
	if err != nil {
		return ""
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var device string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "DEVICE=") {
			device = strings.TrimPrefix(line, "DEVICE=")
			device = strings.Trim(device, "\"'")
		}
	}
	return device
}
