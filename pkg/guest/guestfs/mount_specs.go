//go:build linux

package guestfs

import (
	"log/slog"
	"sort"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

type gfsMountSpec struct {
	Device     string
	GuestMount string
}

func (b *Backend) effectiveMountSpecs() []gfsMountSpec {
	specs := b.mountSpecs
	if len(specs) == 0 {
		specs = mountSpecsFromDiskInfos(b.diskInfos)
	}
	if !b.inspectDone {
		b.inspectDone = true
		if extra := b.discoverMountpoints(); len(extra) > 0 {
			specs = mergeMountSpecs(specs, extra)
			b.mountSpecs = specs
		}
	}
	return sortedMountSpecs(specs)
}

// discoverMountpoints uses guestfish inspect-os + inspect-get-mountpoints
// to discover the guest's full mount table. This catches /boot, /boot/efi,
// and other separate partitions that might not be in diskInfos.
func (b *Backend) discoverMountpoints() []gfsMountSpec {
	if b.session == nil {
		return nil
	}
	out, err := b.session.remoteScript("-inspect-os\n")
	if err != nil {
		slog.Debug("inspect-os unavailable", "error", err)
		return nil
	}
	var root string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "/dev/") {
			root = line
			break
		}
	}
	if root == "" {
		return nil
	}

	out, err = b.session.remoteScript("-inspect-get-mountpoints " + quoteGuestfish(root) + "\n")
	if err != nil {
		slog.Debug("inspect-get-mountpoints failed", "root", root, "error", err)
		return nil
	}

	var specs []gfsMountSpec
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		mp, dev, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		mp = strings.TrimSpace(mp)
		dev = strings.TrimSpace(dev)
		if mp == "" || dev == "" || !strings.HasPrefix(dev, "/dev/") {
			continue
		}
		specs = append(specs, gfsMountSpec{Device: dev, GuestMount: mp})
	}
	if len(specs) > 0 {
		slog.Info("discovered mount points via inspection", "count", len(specs))
	}
	return specs
}

// mergeMountSpecs merges extra mount specs into base, adding only entries
// whose GuestMount path is not already present in base.
func mergeMountSpecs(base, extra []gfsMountSpec) []gfsMountSpec {
	have := make(map[string]bool, len(base))
	for _, s := range base {
		have[s.GuestMount] = true
	}
	merged := append([]gfsMountSpec{}, base...)
	for _, s := range extra {
		if !have[s.GuestMount] {
			merged = append(merged, s)
			have[s.GuestMount] = true
			slog.Debug("adding inspected mount", "device", s.Device, "mountpoint", s.GuestMount)
		}
	}
	return merged
}

func mountSpecsFromDiskInfos(disks []types.DiskInfo) []gfsMountSpec {
	var specs []gfsMountSpec
	for _, d := range disks {
		for _, p := range d.Partitions {
			if p.DevicePath == "" || p.FSType == "" || p.FSType == "swap" {
				continue
			}
			mp := p.MountPoint
			if mp == "" {
				continue
			}
			specs = append(specs, gfsMountSpec{Device: p.DevicePath, GuestMount: mp})
		}
	}
	if len(specs) == 0 {
		for _, d := range disks {
			for _, p := range d.Partitions {
				if p.DevicePath != "" && p.FSType != "" && p.FSType != "swap" {
					return []gfsMountSpec{{Device: p.DevicePath, GuestMount: "/"}}
				}
			}
		}
	}
	return specs
}

func sortedMountSpecs(specs []gfsMountSpec) []gfsMountSpec {
	out := append([]gfsMountSpec{}, specs...)
	sort.SliceStable(out, func(i, j int) bool {
		return len(out[i].GuestMount) < len(out[j].GuestMount)
	})
	return out
}

func mountScriptPrefix(specs []gfsMountSpec) string {
	var script strings.Builder
	script.WriteString("-umount-all\n")
	for _, ms := range sortedMountSpecs(specs) {
		script.WriteString("-mount ")
		script.WriteString(quoteGuestfish(ms.Device))
		script.WriteByte(' ')
		script.WriteString(quoteGuestfish(ms.GuestMount))
		script.WriteByte('\n')
	}
	return script.String()
}
