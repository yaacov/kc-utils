//go:build linux

package guest

import "github.com/yaacov/kc-utils/pkg/common/types"

// WithRootMount ensures disks include a "/" MountPoint for rootDevice.
//
// Prepare records partition mount points on DiskInfo.Partitions, but the OS
// root is often an LVM LV that is not a partition. When that happens, convert
// and finalize would rebuild a mount table missing "/", and guestfish ops
// against /etc and /lib fail (or kill the shared --listen session).
//
// When rootDevice is set and some other device already claims "/", that entry
// is rewritten to rootDevice so remount follows the selected root.
func WithRootMount(disks []types.DiskInfo, rootDevice string) []types.DiskInfo {
	if rootDevice == "" || len(disks) == 0 {
		return disks
	}
	rootDevice = normalizeGuestPath(rootDevice)

	for _, d := range disks {
		for _, p := range d.Partitions {
			if p.DevicePath == rootDevice && p.MountPoint == "/" {
				return disks
			}
		}
	}

	out := append([]types.DiskInfo(nil), disks...)
	for di := range out {
		out[di].Partitions = append([]types.PartitionInfo(nil), out[di].Partitions...)
	}

	rootIdxDisk, rootIdxPart := -1, -1
	for di := range out {
		for pi := range out[di].Partitions {
			p := &out[di].Partitions[pi]
			if p.MountPoint == "/" {
				p.MountPoint = ""
			}
			if p.DevicePath == rootDevice {
				rootIdxDisk, rootIdxPart = di, pi
			}
		}
	}

	if rootIdxDisk >= 0 {
		out[rootIdxDisk].Partitions[rootIdxPart].MountPoint = "/"
		if out[rootIdxDisk].Partitions[rootIdxPart].FSType == "" {
			out[rootIdxDisk].Partitions[rootIdxPart].FSType = "auto"
		}
		return out
	}

	out[0].Partitions = append(out[0].Partitions, types.PartitionInfo{
		DevicePath: rootDevice,
		MountPoint: "/",
		FSType:     "auto",
	})
	return out
}
