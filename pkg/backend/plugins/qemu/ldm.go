//go:build unix

package qemu

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

// activateLDM assembles Windows LDM (dynamic disk) volumes inside the appliance.
// No-op when the guest has no LDM metadata.
func (b *Backend) activateLDM() {
	if b.session == nil || b.session.client == nil {
		return
	}
	out, err := b.session.client.run("ldmtool", "create", "all")
	if err != nil {
		// ldmtool exits non-zero when no LDM disk groups are present.
		slog.Debug("ldmtool create all", "output", strings.TrimSpace(string(out)), "error", err)
		return
	}
	b.ldmActive = true
	slog.Info("LDM volumes activated")
}

// removeLDM tears down LDM device-mapper nodes (best-effort).
func (b *Backend) removeLDM() {
	if !b.ldmActive || b.session == nil || b.session.client == nil {
		return
	}
	if _, err := b.session.client.run("ldmtool", "remove", "all"); err != nil {
		slog.Debug("ldmtool remove all", "error", err)
	}
	b.ldmActive = false
	b.ldmPaths = nil
}

// discoverLDMVolumes lists block devices for activated LDM volumes under /dev/mapper.
func (b *Backend) discoverLDMVolumes() ([]types.PartitionInfo, []string, error) {
	if b.session == nil || b.session.client == nil {
		return nil, nil, nil
	}
	out, err := b.session.client.run("lsblk", "-J", "-l", "-b", "-o", "NAME,PATH,TYPE,FSTYPE,SIZE")
	if err != nil {
		return nil, nil, fmt.Errorf("lsblk -l: %w", err)
	}
	parts, paths := parseLsblkLDMDevices(out)
	for i := range parts {
		if parts[i].FSType == "" {
			parts[i].FSType = b.blkidType(parts[i].DevicePath)
		}
	}
	return parts, paths, nil
}

// parseLsblkLDMDevices extracts LDM mapper devices from lsblk JSON output.
func parseLsblkLDMDevices(out []byte) ([]types.PartitionInfo, []string) {
	var parsed lsblkOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, nil
	}
	var parts []types.PartitionInfo
	var paths []string
	idx := 0
	for _, dev := range parsed.BlockDevices {
		if !isLDMMapperName(dev.Name) {
			continue
		}
		idx++
		parts = append(parts, types.PartitionInfo{
			Index:      idx,
			DevicePath: dev.Path,
			FSType:     dev.FSType,
			SizeBytes:  dev.Size,
		})
		paths = append(paths, dev.Path)
	}
	return parts, paths
}

func isLDMMapperName(name string) bool {
	name = strings.TrimSpace(name)
	return strings.HasPrefix(name, "ldm_")
}

func (b *Backend) mergeLDMIntoDiskInfos(parts []types.PartitionInfo) {
	if len(parts) == 0 {
		return
	}
	if len(b.diskInfos) == 0 {
		b.diskInfos = []types.DiskInfo{{Partitions: parts}}
		return
	}
	b.diskInfos[0].Partitions = append(b.diskInfos[0].Partitions, parts...)
}

func (b *Backend) refreshLDMInDiskInfos(parts []types.PartitionInfo) {
	b.ldmPaths = nil
	for _, p := range parts {
		b.ldmPaths = append(b.ldmPaths, p.DevicePath)
	}
	if len(b.diskInfos) == 0 {
		b.mergeLDMIntoDiskInfos(parts)
		return
	}
	filtered := b.diskInfos[0].Partitions[:0]
	for _, p := range b.diskInfos[0].Partitions {
		if isLDMMapperPath(p.DevicePath) {
			continue
		}
		filtered = append(filtered, p)
	}
	filtered = append(filtered, parts...)
	b.diskInfos[0].Partitions = filtered
}

func isLDMMapperPath(path string) bool {
	return strings.Contains(path, "/mapper/ldm_")
}
