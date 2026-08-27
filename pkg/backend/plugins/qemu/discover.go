//go:build unix

package qemu

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

// lsblkDevice mirrors the JSON emitted by `lsblk -J`.
type lsblkDevice struct {
	Name     string        `json:"name"`
	Path     string        `json:"path"`
	Type     string        `json:"type"`
	FSType   string        `json:"fstype"`
	Size     int64         `json:"size"`
	Children []lsblkDevice `json:"children"`
}

type lsblkOutput struct {
	BlockDevices []lsblkDevice `json:"blockdevices"`
}

// discoverPartitions lists the partitions of an appliance disk device via
// lsblk. A whole-disk filesystem (no partition table) is reported as a single
// partition pointing at the device itself.
func (b *Backend) discoverPartitions(device string) ([]types.PartitionInfo, error) {
	out, err := b.session.client.run("lsblk", "-J", "-b", "-o", "NAME,PATH,TYPE,FSTYPE,SIZE", device)
	if err != nil {
		return nil, fmt.Errorf("lsblk %s: %w", device, err)
	}
	parts, err := parseLsblkPartitions(out)
	if err != nil {
		return nil, err
	}

	// lsblk reads FSTYPE from the udev database, which the minimal appliance does
	// not populate (no udev). Probe any missing type directly with blkid, which
	// reads the on-disk superblock.
	for i := range parts {
		if parts[i].FSType == "" {
			parts[i].FSType = b.blkidType(parts[i].DevicePath)
		}
	}

	// A whole-disk filesystem (no partition table) that lsblk could not type is
	// invisible to parseLsblkPartitions; probe the device itself. Take its size
	// from the top-level lsblk device so the partition is not reported as 0 bytes.
	if len(parts) == 0 {
		if fs := b.blkidType(device); fs != "" {
			parts = append(parts, types.PartitionInfo{
				Index:      1,
				DevicePath: device,
				FSType:     fs,
				SizeBytes:  lsblkTopSize(out),
			})
		}
	}
	return parts, nil
}

// lsblkTopSize returns the size in bytes of the top-level block device in
// `lsblk -J -b` output, or 0 if it cannot be determined. Pure and testable.
func lsblkTopSize(out []byte) int64 {
	var parsed lsblkOutput
	if err := json.Unmarshal(out, &parsed); err != nil || len(parsed.BlockDevices) == 0 {
		return 0
	}
	return parsed.BlockDevices[0].Size
}

// blkidType returns the on-disk filesystem type of device via a direct blkid
// superblock probe, or "" if none is detected. Unlike lsblk's FSTYPE column it
// does not depend on a populated udev database.
func (b *Backend) blkidType(device string) string {
	out, err := b.session.client.run("blkid", "-o", "value", "-s", "TYPE", device)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// parseLsblkPartitions extracts partitions from `lsblk -J` output. A whole-disk
// filesystem (no partition table) is reported as a single partition pointing at
// the device itself. It is pure so it can be unit-tested without a live agent.
func parseLsblkPartitions(out []byte) ([]types.PartitionInfo, error) {
	var parsed lsblkOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("parse lsblk output: %w", err)
	}
	if len(parsed.BlockDevices) == 0 {
		return nil, nil
	}
	top := parsed.BlockDevices[0]

	var parts []types.PartitionInfo
	for _, child := range top.Children {
		if child.Type != "part" {
			continue
		}
		parts = append(parts, types.PartitionInfo{
			Index:      len(parts) + 1,
			DevicePath: child.Path,
			FSType:     child.FSType,
			SizeBytes:  child.Size,
		})
	}
	if len(parts) == 0 && top.FSType != "" {
		// Whole-disk filesystem: no partition table.
		parts = append(parts, types.PartitionInfo{
			Index:      1,
			DevicePath: top.Path,
			FSType:     top.FSType,
			SizeBytes:  top.Size,
		})
	}
	return parts, nil
}

// discoverLVs scans and activates LVM volume groups, returning the paths of the
// activated logical volumes. Scan/activate failures are non-fatal (a guest may
// have no LVM); only the final listing error is returned.
func (b *Backend) discoverLVs() ([]string, error) {
	// Best-effort scan + activate; a guest without LVM makes these no-ops.
	_, _ = b.session.client.run("pvscan", "--cache")
	_, _ = b.session.client.run("vgchange", "-ay")

	out, err := b.session.client.run("lvs", "--noheadings", "-o", "lv_path")
	if err != nil {
		return nil, err
	}
	return parseLVPaths(out), nil
}

// parseLVPaths extracts trimmed, non-empty logical-volume paths from the output
// of `lvs --noheadings -o lv_path`. Pure, so it is unit-testable.
func parseLVPaths(out []byte) []string {
	var lvs []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lvs = append(lvs, line)
	}
	return lvs
}

// remountFromDiskInfos re-establishes mounts recorded by a prior stage when a
// fresh (non-shared) appliance is booted in NewMounted. Mounts are applied
// shortest-mountpoint-first so parents precede children.
func (b *Backend) remountFromDiskInfos() error {
	for _, p := range orderMountPlans(b.diskInfos) {
		hostMountPoint := hostMountFromGuest(b.mountRoot, p.mount)
		if !pathUnderRoot(b.mountRoot, hostMountPoint) {
			return fmt.Errorf("remount %s: host path %s escapes mount root %s", p.mount, hostMountPoint, b.mountRoot)
		}
		if err := b.Mount(p.device, hostMountPoint, "", false); err != nil {
			return fmt.Errorf("remount %s at %s: %w", p.device, p.mount, err)
		}
	}
	return nil
}

// recordAdoptedMounts fills b.mounts from prior-stage disk infos when this
// process adopted a shared appliance and never called Mount itself.
func (b *Backend) recordAdoptedMounts() {
	if len(b.mounts) > 0 {
		return
	}
	b.mounts = mountEntriesFromDiskInfos(b.mountRoot, b.diskInfos)
}

// mountEntriesFromDiskInfos maps recorded guest mounts to appliance paths in
// parent-before-child order (UnmountAll reverses it).
func mountEntriesFromDiskInfos(mountRoot string, diskInfos []types.DiskInfo) []mountEntry {
	var out []mountEntry
	for _, p := range orderMountPlans(diskInfos) {
		host := hostMountFromGuest(mountRoot, p.mount)
		if !pathUnderRoot(mountRoot, host) {
			continue
		}
		app := applianceMountPath(mountRoot, host)
		if app != applianceMountRoot && !pathUnderRoot(applianceMountRoot, app) {
			continue
		}
		out = append(out, mountEntry{
			Device:        p.device,
			AppliancePath: app,
		})
	}
	return out
}

// mountPlan is one recorded (device, guest mount point) pair to re-establish.
type mountPlan struct {
	device string
	mount  string
}

// orderMountPlans selects the mountable partitions from prior-stage disk infos
// and orders them shortest-mountpoint-first so parents mount before children.
// Swap and unmounted/unformatted partitions are skipped. Pure and testable.
func orderMountPlans(diskInfos []types.DiskInfo) []mountPlan {
	var plans []mountPlan
	for _, di := range diskInfos {
		for _, p := range di.Partitions {
			if p.MountPoint == "" || p.DevicePath == "" || p.FSType == "" || p.FSType == "swap" {
				continue
			}
			plans = append(plans, mountPlan{device: p.DevicePath, mount: p.MountPoint})
		}
	}
	sort.SliceStable(plans, func(i, j int) bool {
		return len(plans[i].mount) < len(plans[j].mount)
	})
	return plans
}
