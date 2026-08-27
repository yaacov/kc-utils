//go:build unix

package mount

import (
	"fmt"
	"log/slog"
	"sort"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest"
)

// Apply mounts specs in path-length order and updates disk partition mount points.
func Apply(g *guest.Guest, specs []MountSpec, disks []types.DiskInfo) error {
	for _, spec := range SortSpecs(specs) {
		slog.Info("mounting filesystem",
			"device", spec.DevicePath,
			"mountpoint", spec.GuestMP,
			"fstype", spec.FSType,
			"readOnly", spec.ReadOnly,
		)
		if err := g.MountPartition(spec.DevicePath, spec.GuestMP, spec.ReadOnly); err != nil {
			return fmt.Errorf("mount %s at %s: %w", spec.DevicePath, spec.GuestMP, err)
		}
		updateMountPoint(disks, spec.DevicePath, spec.GuestMP, spec.FSType)
	}
	return nil
}

// updateMountPoint records guestMP on the matching partition. LVM volumes and
// other non-partition devices are appended as synthetic PartitionInfo entries
// so convert/finalize can rebuild the full mount table from prepare-out JSON.
func updateMountPoint(disks []types.DiskInfo, devicePath, guestMP, fsType string) {
	for di := range disks {
		for pi := range disks[di].Partitions {
			if disks[di].Partitions[pi].DevicePath == devicePath {
				disks[di].Partitions[pi].MountPoint = guestMP
				if fsType != "" && disks[di].Partitions[pi].FSType == "" {
					disks[di].Partitions[pi].FSType = fsType
				}
				return
			}
		}
	}
	if len(disks) == 0 || devicePath == "" || guestMP == "" {
		return
	}
	disks[0].Partitions = append(disks[0].Partitions, types.PartitionInfo{
		DevicePath: devicePath,
		MountPoint: guestMP,
		FSType:     fsType,
	})
}

// SortSpecs returns specs sorted by guest mount path length.
func SortSpecs(specs []MountSpec) []MountSpec {
	out := append([]MountSpec{}, specs...)
	sort.Slice(out, func(i, j int) bool {
		return len(out[i].GuestMP) < len(out[j].GuestMP)
	})
	return out
}
