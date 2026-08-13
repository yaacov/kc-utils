//go:build linux

package networkd

import (
	"fmt"
	"log/slog"
	"net"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-linux/systemd"
	"github.com/yaacov/kc-utils/pkg/guest"
)

const (
	virtioNetworkFile   = "10-kc-virtio.network"
	staticNetworkPrefix = "10-kc-static-"
	waitOnlineDropIn    = "kc-timeout.conf"
)

const virtioDHCPConfig = `[Match]
Name=enp*
Driver=virtio_net

[Network]
DHCP=yes

[Match]
Name=eth*
Driver=virtio_net

[Network]
DHCP=yes
`

const waitOnlineDropInContent = `[Service]
ExecStart=
ExecStart=/usr/lib/systemd/systemd-networkd-wait-online --timeout=30 --any
`

// Detect reports whether the guest uses systemd-networkd as its primary network stack.
func Detect(guestRoot string) bool {
	if isNetworkStackAmbiguous(guestRoot) {
		return false
	}
	if guest.FileExists(filepath.Join(guestRoot, "usr", "lib", "systemd", "network", "80-ec2.network")) {
		return true
	}
	if isAmazonLinux2023(guestRoot) {
		return true
	}
	return isNetworkdPrimary(guestRoot)
}

// isNetworkStackAmbiguous is true when both systemd-networkd and NetworkManager
// are enabled. Tier-2 distro shortcuts must not override this conflict.
func isNetworkStackAmbiguous(guestRoot string) bool {
	if !systemd.UnitWantsEnabled(guestRoot, "systemd-networkd.service") {
		return false
	}
	if systemd.UnitIsMasked(guestRoot, "NetworkManager.service") {
		return false
	}
	return systemd.UnitWantsEnabled(guestRoot, "NetworkManager.service")
}

// isAmazonLinux2023 reports whether os-release identifies Amazon Linux 2023,
// which uses systemd-networkd by default. Amazon Linux 2 does not, so it
// falls through to isNetworkdPrimary instead of being matched unconditionally.
func isAmazonLinux2023(guestRoot string) bool {
	for _, rel := range []string{"etc/os-release", "usr/lib/os-release"} {
		data, err := guest.FileRead(filepath.Join(guestRoot, rel))
		if err != nil {
			continue
		}
		values := parseOSRelease(string(data))
		if values["ID"] == "amzn" && values["VERSION_ID"] == "2023" {
			return true
		}
	}
	return false
}

// parseOSRelease parses KEY=VALUE lines from os-release content, unquoting values.
func parseOSRelease(data string) map[string]string {
	values := make(map[string]string)
	for _, line := range strings.Split(data, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		values[key] = strings.Trim(value, `"'`)
	}
	return values
}

// isNetworkdPrimary is true when systemd-networkd is enabled and NetworkManager is not active.
func isNetworkdPrimary(guestRoot string) bool {
	if !systemd.UnitWantsEnabled(guestRoot, "systemd-networkd.service") {
		return false
	}
	if systemd.UnitIsMasked(guestRoot, "NetworkManager.service") {
		return true
	}
	return !systemd.UnitWantsEnabled(guestRoot, "NetworkManager.service")
}

// InstallKubeVirtNetworking writes virtio DHCP config and a short wait-online drop-in.
func InstallKubeVirtNetworking(guestRoot string) error {
	if err := InstallDHCP(guestRoot); err != nil {
		return err
	}
	return InstallWaitOnlineDropIn(guestRoot)
}

// InstallDHCP writes a persistent DHCP profile for virtio NICs on KubeVirt.
func InstallDHCP(guestRoot string) error {
	netDir := filepath.Join(guestRoot, "etc", "systemd", "network")
	if err := guest.FileMkdirAll(netDir, 0o755); err != nil {
		return fmt.Errorf("creating systemd network dir: %w", err)
	}
	path := filepath.Join(netDir, virtioNetworkFile)
	if err := guest.FileWrite(path, []byte(virtioDHCPConfig), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", virtioNetworkFile, err)
	}
	slog.Info("installed virtio DHCP network config", "path", path)
	return nil
}

// WriteStaticNetworks writes MAC-matched static IP profiles for systemd-networkd guests.
func WriteStaticNetworks(guestRoot string, ips []types.StaticIP) error {
	if len(ips) == 0 {
		return nil
	}
	netDir := filepath.Join(guestRoot, "etc", "systemd", "network")
	if err := guest.FileMkdirAll(netDir, 0o755); err != nil {
		return fmt.Errorf("creating systemd network dir: %w", err)
	}
	for i := range ips {
		content, err := staticNetworkContent(&ips[i])
		if err != nil {
			return fmt.Errorf("building static network config for %s: %w", ips[i].MAC, err)
		}
		name := staticNetworkPrefix + sanitizeMAC(ips[i].MAC) + ".network"
		path := filepath.Join(netDir, name)
		if err := guest.FileWrite(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", name, err)
		}
		slog.Info("installed static network config", "path", path, "mac", ips[i].MAC)
	}
	return nil
}

func staticNetworkContent(sip *types.StaticIP) (string, error) {
	prefix := "24"
	if sip.Netmask != "" {
		ones, err := netmaskPrefix(sip.Netmask)
		if err != nil {
			return "", err
		}
		prefix = ones
	}
	mac := strings.ToLower(sip.MAC)
	lines := []string{"[Match]", "MACAddress=" + mac, "", "[Network]", fmt.Sprintf("Address=%s/%s", sip.IP, prefix)}
	if sip.Gateway != "" {
		lines = append(lines, "Gateway="+sip.Gateway)
	}
	if len(sip.DNS) > 0 {
		lines = append(lines, "DNS="+strings.Join(sip.DNS, " "))
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func sanitizeMAC(mac string) string {
	return strings.NewReplacer(":", "-", ".", "-").Replace(strings.ToLower(mac))
}

func netmaskPrefix(mask string) (string, error) {
	parts := strings.Split(mask, ".")
	if len(parts) != 4 {
		return "24", fmt.Errorf("invalid netmask")
	}
	octets := make([]byte, 4)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > 255 {
			return "24", fmt.Errorf("invalid netmask octet: %q", p)
		}
		octets[i] = byte(n)
	}
	ones, bits := net.IPMask(octets).Size()
	if bits == 0 {
		return "24", fmt.Errorf("invalid netmask: %s", mask)
	}
	return fmt.Sprintf("%d", ones), nil
}

// InstallWaitOnlineDropIn shortens wait-online and accepts any configured interface.
func InstallWaitOnlineDropIn(guestRoot string) error {
	dropInDir := filepath.Join(guestRoot, "etc", "systemd", "system",
		"systemd-networkd-wait-online.service.d")
	if err := guest.FileMkdirAll(dropInDir, 0o755); err != nil {
		return fmt.Errorf("creating wait-online drop-in dir: %w", err)
	}
	path := filepath.Join(dropInDir, waitOnlineDropIn)
	if err := guest.FileWrite(path, []byte(waitOnlineDropInContent), 0o644); err != nil {
		return fmt.Errorf("writing wait-online drop-in: %w", err)
	}
	slog.Info("installed wait-online drop-in", "path", path)
	return nil
}
