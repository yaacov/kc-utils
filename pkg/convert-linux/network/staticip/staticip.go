//go:build unix

package staticip

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

// MacToIPLine formats a static IP entry for the macToIP mapping file.
func MacToIPLine(sip *types.StaticIP) string {
	prefix := "24"
	if sip.Netmask != "" {
		if ones, err := netmaskPrefix(sip.Netmask); err == nil {
			prefix = ones
		}
	}
	dns := strings.Join(sip.DNS, ",")
	return fmt.Sprintf("%s:ip:%s,%s,%s,%s", sip.MAC, sip.IP, sip.Gateway, prefix, dns)
}

// WriteMacToIP writes the macToIP mapping file into the guest root.
func WriteMacToIP(guestRoot string, ips []types.StaticIP) error {
	if len(ips) == 0 {
		return nil
	}
	var lines []string
	for i := range ips {
		lines = append(lines, MacToIPLine(&ips[i]))
	}
	content := strings.Join(lines, "\n") + "\n"
	path := filepath.Join(guestRoot, "tmp", "macToIP")
	if err := guestio.FileMkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return guestio.FileWrite(path, []byte(content), 0o644)
}

// FirstbootCommands returns shell commands to configure static IPs on first boot.
func FirstbootCommands() []string {
	return []string{
		`[ -f /tmp/macToIP ] || exit 0`,
		`while IFS= read -r line; do
  mac="${line%%:ip:*}"
  rest="${line#*:ip:}"
  IFS=',' read -r ip gw prefix dns <<< "$rest"
  iface=$(ip -o link | awk -v m="${mac//:/}" 'tolower($0 ~ m) {print $2; exit}' | tr -d ':')
  [ -z "$iface" ] && continue
  if command -v nmcli >/dev/null 2>&1; then
    nmcli con delete "kc-static-$iface" 2>/dev/null || true
    nmcli con add type ethernet ifname "$iface" con-name "kc-static-$iface" ipv4.method manual ipv4.addresses "${ip}/${prefix}" ${gw:+ipv4.gateway "$gw"} ${dns:+ipv4.dns "$dns"}
    nmcli con up "kc-static-$iface"
  else
    ip addr flush dev "$iface"
    ip addr add "${ip}/${prefix}" dev "$iface"
    [ -n "$gw" ] && ip route replace default via "$gw" dev "$iface"
  fi
done < /tmp/macToIP`,
	}
}

func netmaskPrefix(mask string) (string, error) {
	parts := strings.Split(mask, ".")
	if len(parts) != 4 {
		return "24", fmt.Errorf("invalid netmask")
	}
	var bits int
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return "24", fmt.Errorf("invalid netmask octet: %w", err)
		}
		for i := 7; i >= 0; i-- {
			if n&(1<<i) != 0 {
				bits++
			}
		}
	}
	return fmt.Sprintf("%d", bits), nil
}
