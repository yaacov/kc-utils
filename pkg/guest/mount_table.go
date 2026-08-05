//go:build linux

package guest

import "github.com/yaacov/kc-utils/pkg/common/types"

// WithRootMount ensures disks include a "/" MountPoint for rootDevice.
//
// Prepare records partition mount points on DiskInfo.Partitions, but the OS
// root is often an LVM LV that is not a partition. When that happens, convert
// and finalize would rebuild a mount table missing "/", and guestfish ops
// against /etc and /lib fail (or kill the shared --listen session).
func WithRootMount(disks []types.DiskInfo, rootDevice string) []types.DiskInfo {
	if rootDevice == "" || len(disks) == 0 {
		return disks
	}
	rootDevice = normalizeGuestPath(rootDevice)
	for _, d := range disks {
		for _, p := range d.Partitions {
			if p.MountPoint == "/" || p.DevicePath == rootDevice {
				return disks
			}
		}
	}
	out := append([]types.DiskInfo(nil), disks...)
	out[0].Partitions = append(append([]types.PartitionInfo(nil), out[0].Partitions...), types.PartitionInfo{
		DevicePath: rootDevice,
		MountPoint: "/",
		FSType:     "auto",
	})
	return out
}
