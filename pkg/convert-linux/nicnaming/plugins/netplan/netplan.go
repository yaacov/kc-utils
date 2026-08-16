//go:build unix

package netplan

import (
	"bufio"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type Plugin struct{}

func init() {
	nicnaming.Namers.Register("netplan", &Plugin{})
}

func (p *Plugin) Detect(guestRoot string) bool {
	dir := filepath.Join(guestRoot, "etc", "netplan")
	return guest.FileIsDir(dir)
}

func (p *Plugin) ResolveNames(guestRoot string, entries []nicnaming.MacIPEntry) ([]nicnaming.NamingRule, error) {
	netplanDir := filepath.Join(guestRoot, "etc", "netplan")
	yamlFiles, err := guest.FileGlob(filepath.Join(netplanDir, "*.yaml"))
	if err != nil {
		return nil, nil
	}
	ymlFiles, _ := guest.FileGlob(filepath.Join(netplanDir, "*.yml"))
	yamlFiles = append(yamlFiles, ymlFiles...)

	if len(yamlFiles) == 0 {
		return nil, nil
	}

	var rules []nicnaming.NamingRule
	for _, entry := range entries {
		device := findInterfaceInNetplan(yamlFiles, entry.IP)
		if device != "" {
			rules = append(rules, nicnaming.NamingRule{MAC: entry.MAC, Device: device})
		}
	}
	return rules, nil
}

// findInterfaceInNetplan does a simple text scan of netplan YAML files
// to find an interface section containing the target IP. This avoids
// pulling in a YAML parser dependency for a straightforward pattern match.
func findInterfaceInNetplan(files []string, ip string) string {
	for _, path := range files {
		device := scanNetplanFile(path, ip)
		if device != "" {
			return device
		}
	}
	return ""
}

func scanNetplanFile(path, ip string) string {
	data, err := guest.FileRead(path)
	if err != nil {
		return ""
	}

	var currentIface string
	inEthernets := false
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "ethernets:" {
			inEthernets = true
			continue
		}
		if inEthernets && len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
			inEthernets = false
		}
		if !inEthernets {
			continue
		}

		// Interface name lines are indented exactly 4 spaces (or 1 level)
		// and end with ':'
		if strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, "-") {
			indent := len(line) - len(strings.TrimLeft(line, " "))
			if indent <= 4 {
				currentIface = strings.TrimSuffix(trimmed, ":")
			}
		}

		if currentIface != "" && strings.Contains(trimmed, ip) {
			return currentIface
		}
	}
	return ""
}
