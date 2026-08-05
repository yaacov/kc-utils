//go:build linux

package guest

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestWithRootMountAddsLV(t *testing.T) {
	disks := []types.DiskInfo{{
		Path: "/dev/block0",
		Partitions: []types.PartitionInfo{
			{DevicePath: "/dev/sda1", FSType: "vfat", MountPoint: "/boot/efi"},
			{DevicePath: "/dev/sda2", FSType: "xfs", MountPoint: "/boot"},
		},
	}}
	got := WithRootMount(disks, "/dev/rhel/root")
	if len(got[0].Partitions) != 3 {
		t.Fatalf("partitions=%d want 3: %#v", len(got[0].Partitions), got[0].Partitions)
	}
	last := got[0].Partitions[2]
	if last.DevicePath != "/dev/rhel/root" || last.MountPoint != "/" || last.FSType != "auto" {
		t.Fatalf("root entry=%#v", last)
	}
	// Original slice must be unchanged.
	if len(disks[0].Partitions) != 2 {
		t.Fatalf("input mutated: %#v", disks[0].Partitions)
	}
}

func TestWithRootMountIdempotent(t *testing.T) {
	disks := []types.DiskInfo{{
		Partitions: []types.PartitionInfo{
			{DevicePath: "/dev/rhel/root", FSType: "xfs", MountPoint: "/"},
			{DevicePath: "/dev/sda2", FSType: "xfs", MountPoint: "/boot"},
		},
	}}
	got := WithRootMount(disks, "/dev/rhel/root")
	if len(got[0].Partitions) != 2 {
		t.Fatalf("should not duplicate root: %#v", got[0].Partitions)
	}
}

func TestWithRootMountEmpty(t *testing.T) {
	if WithRootMount(nil, "/dev/sda1") != nil {
		t.Fatal("nil disks")
	}
	disks := []types.DiskInfo{{Path: "/dev/sda"}}
	if got := WithRootMount(disks, ""); len(got) != 1 || len(got[0].Partitions) != 0 {
		t.Fatalf("empty rootDevice: %#v", got)
	}
}
