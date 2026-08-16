//go:build unix

package guestfs

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func (b *Backend) discoverPartitions(diskPath string) ([]types.PartitionInfo, error) {
	if err := b.ensureSession(); err != nil {
		return nil, err
	}
	diskDev, err := b.diskDeviceName(diskPath)
	if err != nil {
		return nil, err
	}

	out, err := b.session.remote("list-filesystems")
	if err != nil {
		return nil, fmt.Errorf("list-filesystems: %w", err)
	}
	fsMap := parseListFilesystems(string(out))

	devices := make([]string, 0, len(fsMap))
	for device := range fsMap {
		if deviceBelongsToDisk(device, diskDev) {
			devices = append(devices, device)
		}
	}
	sort.Strings(devices)

	parts := make([]types.PartitionInfo, 0, len(devices))
	for i, device := range devices {
		pi := types.PartitionInfo{
			Index:      i + 1,
			DevicePath: device,
			FSType:     fsMap[device],
		}
		szOut, szErr := b.session.remoteScript(
			"-blockdev-getsize64 " + quoteGuestfish(device) + "\n",
		)
		if szErr == nil {
			if sz, perr := strconv.ParseInt(strings.TrimSpace(string(szOut)), 10, 64); perr == nil {
				pi.SizeBytes = sz
			}
		}
		parts = append(parts, pi)
	}
	return parts, nil
}

func (b *Backend) discoverLVs() ([]string, error) {
	if err := b.ensureSession(); err != nil {
		return nil, err
	}
	out, err := b.session.remoteScript("-lvs\n")
	if err != nil {
		return nil, nil
	}
	var lvs []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "/dev/") {
			line = "/dev/" + line
		}
		lvs = append(lvs, line)
	}
	return lvs, nil
}

func (b *Backend) diskDeviceName(diskPath string) (string, error) {
	idx := -1
	for i, p := range b.diskPaths {
		if p == diskPath {
			idx = i
			break
		}
	}
	if idx < 0 {
		return "", fmt.Errorf("disk %s not in session disk list", diskPath)
	}
	out, err := b.session.remote("list-devices")
	if err != nil {
		return "", fmt.Errorf("list-devices: %w", err)
	}
	var devices []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "/dev/") {
			devices = append(devices, line)
		}
	}
	if idx >= len(devices) {
		return "", fmt.Errorf("list-devices: no device for disk index %d (%s)", idx, diskPath)
	}
	return devices[idx], nil
}

func parseListFilesystems(out string) map[string]string {
	m := make(map[string]string)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		device, fstype, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		device = strings.TrimSpace(device)
		fstype = strings.TrimSpace(fstype)
		if device == "" || fstype == "" {
			continue
		}
		m[device] = fstype
	}
	return m
}

func deviceBelongsToDisk(device, diskDev string) bool {
	if device == diskDev {
		return true
	}
	if !strings.HasPrefix(device, diskDev) {
		return false
	}
	rest := device[len(diskDev):]
	if rest == "" {
		return false
	}
	if rest[0] >= '0' && rest[0] <= '9' {
		return true
	}
	if rest[0] == 'p' && len(rest) > 1 && rest[1] >= '0' && rest[1] <= '9' {
		return true
	}
	return false
}
