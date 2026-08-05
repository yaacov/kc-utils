//go:build linux

package dhclient

import (
	"bufio"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type Plugin struct{}

func init() {
	nicnaming.Namers.Register("dhclient", &Plugin{})
}

func (p *Plugin) Detect(guestRoot string) bool {
	for _, dir := range leaseDirs(guestRoot) {
		if guest.FileIsDir(dir) {
			return true
		}
	}
	return false
}

func (p *Plugin) ResolveNames(guestRoot string, entries []nicnaming.MacIPEntry) ([]nicnaming.NamingRule, error) {
	var leaseFiles []string
	for _, dir := range leaseDirs(guestRoot) {
		matches, _ := guest.FileGlob(filepath.Join(dir, "dhclient-*"))
		leaseFiles = append(leaseFiles, matches...)
	}
	if len(leaseFiles) == 0 {
		return nil, nil
	}

	var rules []nicnaming.NamingRule
	for _, entry := range entries {
		device := findDeviceInLeases(leaseFiles, entry.IP)
		if device != "" {
			rules = append(rules, nicnaming.NamingRule{MAC: entry.MAC, Device: device})
		}
	}
	return rules, nil
}

func leaseDirs(guestRoot string) []string {
	return []string{
		filepath.Join(guestRoot, "var", "lib", "dhclient"),
		filepath.Join(guestRoot, "var", "lib", "NetworkManager"),
	}
}

func findDeviceInLeases(files []string, ip string) string {
	for _, path := range files {
		device := parseLeaseFile(path, ip)
		if device != "" {
			return device
		}
	}
	return ""
}

func parseLeaseFile(path, targetIP string) string {
	data, err := guest.FileRead(path)
	if err != nil {
		return ""
	}

	var currentIface, currentIP string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = strings.TrimSuffix(line, ";")

		switch {
		case strings.HasPrefix(line, "interface"):
			currentIface = extractQuoted(line)
		case strings.HasPrefix(line, "fixed-address"):
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentIP = parts[1]
			}
		case line == "}":
			if currentIP == targetIP && currentIface != "" {
				return currentIface
			}
			currentIface = ""
			currentIP = ""
		}
	}
	return ""
}

func extractQuoted(line string) string {
	start := strings.IndexByte(line, '"')
	if start < 0 {
		return ""
	}
	end := strings.IndexByte(line[start+1:], '"')
	if end < 0 {
		return ""
	}
	return line[start+1 : start+1+end]
}
