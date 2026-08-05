package env

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

// ParseStaticIPs parses V2V_staticIPs into structured entries.
func ParseStaticIPs(raw string) ([]types.StaticIP, error) {
	if raw == "" {
		return nil, nil
	}
	segments := strings.Split(raw, "_")
	var out []types.StaticIP
	for _, seg := range segments {
		if seg == "" {
			continue
		}
		mac, rest, ok := strings.Cut(seg, ":ip:")
		if !ok {
			return nil, fmt.Errorf("invalid static IP segment %q", seg)
		}
		parts := strings.Split(rest, ",")
		if len(parts) < 1 || parts[0] == "" {
			return nil, fmt.Errorf("invalid static IP segment %q", seg)
		}
		sip := types.StaticIP{MAC: mac, IP: parts[0]}
		if len(parts) > 1 && parts[1] != "" {
			sip.Gateway = parts[1]
		}
		if len(parts) > 2 && parts[2] != "" {
			if prefix, err := strconv.Atoi(parts[2]); err == nil {
				sip.Netmask = prefixLengthToNetmask(prefix)
			}
		}
		if len(parts) > 3 && parts[3] != "" {
			sip.DNS = parts[3:]
		}
		out = append(out, sip)
	}
	return out, nil
}

func prefixLengthToNetmask(prefix int) string {
	if prefix <= 0 || prefix > 32 {
		return "255.255.255.0"
	}
	mask := net.CIDRMask(prefix, 32)
	return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
}
