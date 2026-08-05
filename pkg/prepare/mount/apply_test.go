//go:build linux

package mount

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestUpdateMountPointPartition(t *testing.T) {
	disks := []types.DiskInfo{{
		Partitions: []types.PartitionInfo{
			{DevicePath: "/dev/sda1", FSType: "xfs"},
		},
	}}
	updateMountPoint(disks, "/dev/sda1", "/boot", "xfs")
	if disks[0].Partitions[0].MountPoint != "/boot" {
		t.Fatalf("%#v", disks[0].Partitions[0])
	}
}

func TestUpdateMountPointAppendsLV(t *testing.T) {
	disks := []types.DiskInfo{{
		Partitions: []types.PartitionInfo{
			{DevicePath: "/dev/sda1", FSType: "vfat", MountPoint: "/boot/efi"},
		},
	}}
	updateMountPoint(disks, "/dev/rhel/root", "/", "xfs")
	if len(disks[0].Partitions) != 2 {
		t.Fatalf("got %d partitions", len(disks[0].Partitions))
	}
	lv := disks[0].Partitions[1]
	if lv.DevicePath != "/dev/rhel/root" || lv.MountPoint != "/" || lv.FSType != "xfs" {
		t.Fatalf("%#v", lv)
	}
}
