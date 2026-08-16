//go:build unix

package root

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest"
	"github.com/yaacov/kc-utils/pkg/prepare/inspect"
)

// Discover finds OS root candidates on all guest partitions and LVM volumes.
func Discover(g *guest.Guest, disks []types.DiskInfo, lvPaths []string) ([]types.RootCandidate, error) {
	var candidates []types.RootCandidate

	for di, d := range disks {
		for _, p := range d.Partitions {
			if skipFSType(p.FSType) {
				continue
			}
			c, ok, err := probeDevice(g, p.DevicePath, di, p.Index, p.FSType)
			if err != nil {
				slog.Warn("root probe failed", "device", p.DevicePath, "error", err)
				continue
			}
			if ok {
				candidates = append(candidates, c)
			}
		}
	}

	for _, lvPath := range lvPaths {
		ft, err := g.FSType(lvPath)
		if err != nil || skipFSType(ft) {
			continue
		}
		c, ok, err := probeDevice(g, lvPath, 0, 0, ft)
		if err != nil {
			slog.Warn("root probe failed", "device", lvPath, "error", err)
			continue
		}
		if ok {
			candidates = append(candidates, c)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no root device found in guest")
	}
	return candidates, nil
}

func skipFSType(ft string) bool {
	ft = strings.ToLower(strings.TrimSpace(ft))
	switch {
	case ft == "" || ft == "unknown" || ft == "swap":
		return true
	case strings.HasPrefix(ft, "swap") || strings.Contains(ft, "swap"):
		// guestfish may report "linux-swap", "linux-swap(v1)", etc.
		return true
	case ft == "crypto_luks" || strings.Contains(ft, "crypto_luks") || ft == "bitlocker":
		// Locked containers are not mountable roots; unlocked mappers are probed separately.
		return true
	case ft == "lvm2_member" || strings.Contains(ft, "lvm2_member"):
		// LVM physical volumes are not mountable; their logical volumes are
		// discovered and probed separately via LVPaths.
		return true
	default:
		return false
	}
}

func probeDevice(g *guest.Guest, device string, diskIndex, partIndex int, ft string) (types.RootCandidate, bool, error) {
	scanHost, err := g.ProbeMount(device, ft)
	if err != nil {
		return types.RootCandidate{}, false, err
	}
	defer func() {
		if uerr := g.ProbeUnmount(scanHost); uerr != nil {
			slog.Warn("unmount scan path failed", "path", scanHost, "error", uerr)
		}
	}()

	data, ok := inspect.ProbeRoot(scanHost)
	if !ok {
		slog.Info("root probe miss",
			"device", device,
			"fstype", ft,
			"diskIndex", diskIndex,
			"partIndex", partIndex,
			"scanPath", scanHost,
		)
		return types.RootCandidate{}, false, nil
	}
	slog.Info("root probe hit",
		"device", device,
		"fstype", ft,
		"os", data.Type,
		"distro", data.Distro,
		"product", inspect.ProductName(data),
	)
	return types.RootCandidate{
		DevicePath:  device,
		DiskIndex:   diskIndex,
		PartIndex:   partIndex,
		FSType:      ft,
		Inspect:     *data,
		ProductName: inspect.ProductName(data),
	}, true, nil
}
