//go:build linux

package nicnaming

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/plugin"
	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest"
)

// NamingRule represents a udev rule that pins a MAC address to an interface name.
type NamingRule struct {
	MAC    string
	Device string
}

// MacIPEntry maps a MAC address to an IP for lookup by NICNamer plugins.
type MacIPEntry struct {
	MAC string
	IP  string
}

// NICNamer discovers MAC-to-interface-name bindings from a specific network
// config backend on the mounted guest filesystem.
type NICNamer interface {
	// Detect returns true if this backend is present on the guest.
	Detect(guestRoot string) bool
	// ResolveNames returns naming rules from the guest network config.
	ResolveNames(guestRoot string, entries []MacIPEntry) ([]NamingRule, error)
}

// Namers is the global registry for NIC naming plugins.
var Namers = plugin.NewRegistry[string, NICNamer]()

// Apply discovers MAC-to-name bindings and writes udev rules + systemd .link files.
func Apply(guestRoot string, staticIPs []types.StaticIP) error {
	if len(staticIPs) == 0 {
		return nil
	}

	entries := make([]MacIPEntry, 0, len(staticIPs))
	for _, sip := range staticIPs {
		entries = append(entries, MacIPEntry{MAC: sip.MAC, IP: sip.IP})
	}

	var allRules []NamingRule
	for name, namer := range Namers.All() {
		if !namer.Detect(guestRoot) {
			continue
		}
		rules, err := namer.ResolveNames(guestRoot, entries)
		if err != nil {
			slog.Warn("NIC namer failed", "namer", name, "error", err)
			continue
		}
		if len(rules) > 0 {
			slog.Debug("NIC namer resolved names", "namer", name, "count", len(rules))
			allRules = append(allRules, rules...)
		}
	}

	if len(allRules) == 0 {
		slog.Debug("no NIC naming rules resolved")
		return nil
	}

	deduped := deduplicateRules(allRules)
	if len(deduped) == 0 {
		return nil
	}

	if err := writeUdevRules(guestRoot, deduped); err != nil {
		return err
	}

	writeSystemdLinks(guestRoot, deduped)
	return nil
}

// deduplicateRules removes duplicate rules and rejects conflicting ones
// (same MAC mapping to different names).
func deduplicateRules(rules []NamingRule) []NamingRule {
	type ruleKey struct {
		mac    string
		device string
	}
	seen := make(map[ruleKey]bool)
	macToDevice := make(map[string]string)
	var result []NamingRule

	for _, r := range rules {
		macLower := strings.ToLower(r.MAC)
		key := ruleKey{mac: macLower, device: r.Device}
		if seen[key] {
			continue
		}
		if existing, ok := macToDevice[macLower]; ok && existing != r.Device {
			slog.Warn("conflicting NIC naming rules for same MAC",
				"mac", r.MAC, "device1", existing, "device2", r.Device)
			continue
		}
		seen[key] = true
		macToDevice[macLower] = r.Device
		result = append(result, r)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Device < result[j].Device
	})
	return result
}

func writeUdevRules(guestRoot string, rules []NamingRule) error {
	rulesDir := filepath.Join(guestRoot, "etc", "udev", "rules.d")
	if err := guest.FileMkdirAll(rulesDir, 0o755); err != nil {
		return fmt.Errorf("creating udev rules.d: %w", err)
	}

	rulesPath := filepath.Join(rulesDir, "70-persistent-net.rules")

	// Don't overwrite existing non-empty rules
	if data, err := guest.FileRead(rulesPath); err == nil && len(data) > 0 {
		slog.Info("70-persistent-net.rules already exists, skipping")
		return nil
	}

	var lines []string
	for _, r := range rules {
		mac := strings.ToLower(r.MAC)
		line := `SUBSYSTEM=="net",ACTION=="add",ATTR{address}==` +
			`"` + mac + `",NAME="` + r.Device + `"`
		lines = append(lines, line)
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := guest.FileWrite(rulesPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing udev rules: %w", err)
	}
	slog.Info("wrote NIC naming udev rules", "path", rulesPath, "count", len(rules))
	return nil
}

// writeSystemdLinks generates .link files for RHEL 9+ where predictable naming
// overrides raw udev NAME= rules.
func writeSystemdLinks(guestRoot string, rules []NamingRule) {
	linkDir := filepath.Join(guestRoot, "etc", "systemd", "network")
	if err := guest.FileMkdirAll(linkDir, 0o755); err != nil {
		slog.Warn("creating systemd network dir", "error", err)
		return
	}

	for _, r := range rules {
		mac := strings.ToLower(r.MAC)
		content := fmt.Sprintf("[Match]\nMACAddress=%s\n\n[Link]\nName=%s\n", mac, r.Device)
		linkPath := filepath.Join(linkDir, fmt.Sprintf("10-v2v-%s.link", r.Device))
		if err := guest.FileWrite(linkPath, []byte(content), 0o644); err != nil {
			slog.Warn("writing systemd .link file", "device", r.Device, "error", err)
		}
	}
	slog.Info("wrote systemd .link files", "count", len(rules))
}
