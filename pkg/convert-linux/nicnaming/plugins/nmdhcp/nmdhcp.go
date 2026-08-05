//go:build linux

package nmdhcp

import (
	"bufio"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming"
	"github.com/yaacov/kc-utils/pkg/guest"
)

var leaseUUIDRe = regexp.MustCompile(`^.*-([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})-(.+)\.lease$`)

type Plugin struct{}

func init() {
	nicnaming.Namers.Register("nmdhcp", &Plugin{})
}

func (p *Plugin) Detect(guestRoot string) bool {
	dir := filepath.Join(guestRoot, "var", "lib", "NetworkManager")
	return guest.FileIsDir(dir)
}

func (p *Plugin) ResolveNames(guestRoot string, entries []nicnaming.MacIPEntry) ([]nicnaming.NamingRule, error) {
	leaseDir := filepath.Join(guestRoot, "var", "lib", "NetworkManager")
	files, err := guest.FileGlob(filepath.Join(leaseDir, "*.lease"))
	if err != nil || len(files) == 0 {
		return nil, nil
	}

	timestamps := readTimestamps(filepath.Join(leaseDir, "timestamps"))

	var rules []nicnaming.NamingRule
	for _, entry := range entries {
		device := findDeviceFromLeases(files, timestamps, entry.IP)
		if device != "" {
			rules = append(rules, nicnaming.NamingRule{MAC: entry.MAC, Device: device})
		}
	}
	return rules, nil
}

func findDeviceFromLeases(files []string, timestamps map[string]int64, ip string) string {
	type candidate struct {
		iface     string
		timestamp int64
	}
	var candidates []candidate

	for _, f := range files {
		if !leaseContainsAddress(f, ip) {
			continue
		}
		m := leaseUUIDRe.FindStringSubmatch(filepath.Base(f))
		if m == nil {
			continue
		}
		uuid := m[1]
		iface := m[2]
		ts := timestamps[uuid]
		candidates = append(candidates, candidate{iface: iface, timestamp: ts})
	}

	if len(candidates) == 0 {
		return ""
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].timestamp > candidates[j].timestamp
	})
	return candidates[0].iface
}

func leaseContainsAddress(path, ip string) bool {
	data, err := guest.FileRead(path)
	if err != nil {
		return false
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "ADDRESS="+ip {
			return true
		}
	}
	return false
}

func readTimestamps(path string) map[string]int64 {
	result := make(map[string]int64)
	data, err := guest.FileRead(path)
	if err != nil {
		return result
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "[timestamps]" || line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		ts, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if err == nil {
			result[strings.TrimSpace(parts[0])] = ts
		}
	}
	return result
}
