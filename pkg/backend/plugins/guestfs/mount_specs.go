//go:build linux

package guestfs

import (
	"fmt"
	"log/slog"
	"path"
	"sort"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

type gfsMountSpec struct {
	Device     string
	GuestMount string
}

func (b *Backend) effectiveMountSpecs() ([]gfsMountSpec, error) {
	specs := b.mountSpecs
	if len(specs) == 0 {
		specs = mountSpecsFromDiskInfos(b.diskInfos)
	}
	if !b.inspectDone {
		if b.session == nil {
			return sortedMountSpecs(specs), nil
		}
		extra, err := b.discoverMountpoints(preferredRoot(specs))
		if err != nil {
			return nil, err
		}
		b.inspectDone = true
		if len(extra) > 0 {
			specs = mergeMountSpecs(specs, extra)
			b.mountSpecs = specs
		}
	}
	return sortedMountSpecs(specs), nil
}

// discoverMountpoints uses guestfish inspect-os + inspect-get-mountpoints
// to discover the guest's full mount table. This catches /boot, /boot/efi,
// and other separate partitions that might not be in diskInfos.
//
// When preferred is set, the matching inspect-os root is used; when unset,
// the first inspect-os root is the default. A set preferred that matches no
// inspect-os root is an error.
func (b *Backend) discoverMountpoints(preferred string) ([]gfsMountSpec, error) {
	if b.session == nil {
		return nil, nil
	}
	out, err := b.session.remoteScript("-inspect-os\n")
	if err != nil {
		slog.Debug("inspect-os unavailable", "error", err)
		return nil, nil
	}
	roots := parseInspectOSRoots(string(out))
	root, err := selectInspectRoot(roots, preferred)
	if err != nil {
		return nil, err
	}
	if root == "" {
		return nil, nil
	}

	out, err = b.session.remoteScript("-inspect-get-mountpoints " + quoteGuestfish(root) + "\n")
	if err != nil {
		slog.Debug("inspect-get-mountpoints failed", "root", root, "error", err)
		return nil, nil
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
		slog.Info("discovered mount points via inspection", "root", root, "preferred", preferred, "count", len(specs))
	}
	return specs, nil
}

func preferredRoot(specs []gfsMountSpec) string {
	for _, s := range specs {
		if s.GuestMount == "/" && s.Device != "" {
			return s.Device
		}
	}
	return ""
}

func parseInspectOSRoots(out string) []string {
	var roots []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "/dev/") {
			roots = append(roots, line)
		}
	}
	return roots
}

// selectInspectRoot picks which inspect-os root to use for mountpoint discovery.
// Preferred (prepare/user root at "/") wins when it matches; otherwise the first
// inspect-os root is the default when preferred is empty.
func selectInspectRoot(roots []string, preferred string) (string, error) {
	if len(roots) == 0 {
		return "", nil
	}
	preferred = strings.TrimSpace(preferred)
	if preferred == "" {
		return roots[0], nil
	}
	for _, root := range roots {
		if deviceEqual(preferred, root) {
			return root, nil
		}
	}
	return "", fmt.Errorf("preferred root %s not found in inspect-os roots (%s)",
		preferred, strings.Join(roots, ", "))
}

// deviceEqual reports whether two guest block device paths refer to the same
// device, including common LVM path aliases.
func deviceEqual(a, b string) bool {
	a = path.Clean(strings.TrimSpace(a))
	b = path.Clean(strings.TrimSpace(b))
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	if lvmAliasesEqual(a, b) {
		return true
	}
	// Last resort: match by basename when both paths are single-component
	// leaves (e.g. /dev/sda1 vs /dev/mapper/sda1). Excluded for multi-
	// component paths like /dev/VG/LV to avoid false positives.
	if isDevLeaf(a) && isDevLeaf(b) {
		return path.Base(a) == path.Base(b)
	}
	return false
}

func isDevLeaf(dev string) bool {
	if strings.HasPrefix(dev, "/dev/mapper/") {
		rest := strings.TrimPrefix(dev, "/dev/mapper/")
		return rest != "" && !strings.Contains(rest, "/")
	}
	if strings.HasPrefix(dev, "/dev/") {
		rest := strings.TrimPrefix(dev, "/dev/")
		return rest != "" && !strings.Contains(rest, "/")
	}
	return false
}

// lvmAliasesEqual checks whether a and b refer to the same LVM device by
// normalizing /dev/VG/LV paths to /dev/mapper/VG-LV form. When only one
// side is an LVM path, it compares the mapper form against the other's full
// path or basename (handles guestfish returning a short device name).
func lvmAliasesEqual(a, b string) bool {
	ma, oka := mapperAlias(a)
	mb, okb := mapperAlias(b)
	if oka && okb {
		return ma == mb
	}
	if oka && mb == "" {
		return ma == b || ma == path.Base(b)
	}
	if okb && ma == "" {
		return mb == a || mb == path.Base(a)
	}
	return false
}

// mapperAlias returns the /dev/mapper/... form of an LVM path, or the input
// when it is already a mapper path. ok is false when the path is not an LVM
// style device (/dev/VG/LV or /dev/mapper/...).
func mapperAlias(dev string) (string, bool) {
	dev = path.Clean(dev)
	if strings.HasPrefix(dev, "/dev/mapper/") {
		return dev, true
	}
	// /dev/VG/LV (exactly two path components under /dev/)
	rest := strings.TrimPrefix(dev, "/dev/")
	if rest == dev || rest == "" {
		return "", false
	}
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", false
	}
	return "/dev/mapper/" + escapeLVMComponent(parts[0]) + "-" + escapeLVMComponent(parts[1]), true
}

func escapeLVMComponent(s string) string {
	return strings.ReplaceAll(s, "-", "--")
}

// mergeMountSpecs merges extra mount specs into base, adding only entries
// whose GuestMount path is not already present in base. When both sides list
// the same path with devices that are not equal, base wins and a warning is logged.
func mergeMountSpecs(base, extra []gfsMountSpec) []gfsMountSpec {
	have := make(map[string]gfsMountSpec, len(base))
	for _, s := range base {
		have[s.GuestMount] = s
	}
	merged := append([]gfsMountSpec{}, base...)
	for _, s := range extra {
		if existing, ok := have[s.GuestMount]; ok {
			if !deviceEqual(existing.Device, s.Device) {
				slog.Warn("inspect mount disagrees with prepare for path; keeping prepare",
					"mountpoint", s.GuestMount,
					"prepareDevice", existing.Device,
					"inspectDevice", s.Device,
				)
			}
			continue
		}
		merged = append(merged, s)
		have[s.GuestMount] = s
		slog.Debug("adding inspected mount", "device", s.Device, "mountpoint", s.GuestMount)
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
