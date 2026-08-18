//go:build unix

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

func TestWithRootMountRewritesSlash(t *testing.T) {
	disks := []types.DiskInfo{{
		Partitions: []types.PartitionInfo{
			{DevicePath: "/dev/sda2", FSType: "xfs", MountPoint: "/"},
			{DevicePath: "/dev/sda1", FSType: "xfs", MountPoint: "/boot"},
			{DevicePath: "/dev/sda3", FSType: "xfs"},
		},
	}}
	got := WithRootMount(disks, "/dev/sda3")
	var slashDev string
	slashCount := 0
	for _, p := range got[0].Partitions {
		if p.MountPoint == "/" {
			slashCount++
			slashDev = p.DevicePath
		}
	}
	if slashCount != 1 || slashDev != "/dev/sda3" {
		t.Fatalf("want single / on /dev/sda3, got %#v", got[0].Partitions)
	}
	if disks[0].Partitions[0].MountPoint != "/" {
		t.Fatal("input mutated")
	}
}

func TestWithRootMountSetsExistingDevice(t *testing.T) {
	disks := []types.DiskInfo{{
		Partitions: []types.PartitionInfo{
			{DevicePath: "/dev/sda1", FSType: "xfs", MountPoint: "/boot"},
			{DevicePath: "/dev/rhel/root", FSType: "xfs"},
		},
	}}
	got := WithRootMount(disks, "/dev/rhel/root")
	found := false
	for _, p := range got[0].Partitions {
		if p.DevicePath == "/dev/rhel/root" && p.MountPoint == "/" {
			found = true
		}
		if p.MountPoint == "/" && p.DevicePath != "/dev/rhel/root" {
			t.Fatalf("extra /: %#v", p)
		}
	}
	if !found {
		t.Fatalf("root not set: %#v", got[0].Partitions)
	}
}
