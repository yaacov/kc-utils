//go:build linux

package guestfs

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestMountSpecsFromDiskInfosFallback(t *testing.T) {
	disks := []types.DiskInfo{{
		Partitions: []types.PartitionInfo{
			{DevicePath: "/dev/sda1", FSType: "ext4"},
		},
	}}
	specs := mountSpecsFromDiskInfos(disks)
	if len(specs) != 1 {
		t.Fatalf("got %d specs, want 1: %#v", len(specs), specs)
	}
	if specs[0].GuestMount != "/" {
		t.Fatalf("fallback should mount at /, got %q", specs[0].GuestMount)
	}
}

func TestMountSpecsFromDiskInfosEmpty(t *testing.T) {
	specs := mountSpecsFromDiskInfos(nil)
	if len(specs) != 0 {
		t.Fatalf("expected empty specs for nil disks, got %d", len(specs))
	}
}

func TestMountSpecsFromDiskInfosSkipsSwap(t *testing.T) {
	disks := []types.DiskInfo{{
		Partitions: []types.PartitionInfo{
			{DevicePath: "/dev/sda1", FSType: "swap", MountPoint: "none"},
			{DevicePath: "/dev/sda2", FSType: "ext4", MountPoint: "/"},
		},
	}}
	specs := mountSpecsFromDiskInfos(disks)
	if len(specs) != 1 {
		t.Fatalf("got %d specs, want 1 (swap excluded): %#v", len(specs), specs)
	}
	if specs[0].Device != "/dev/sda2" {
		t.Fatalf("expected /dev/sda2, got %q", specs[0].Device)
	}
}

func TestEffectiveMountSpecsUsesExplicit(t *testing.T) {
	explicit := []gfsMountSpec{
		{Device: "/dev/sda1", GuestMount: "/boot"},
		{Device: "/dev/sda2", GuestMount: "/"},
	}
	b := &Backend{mountSpecs: explicit}
	got := b.effectiveMountSpecs()
	if len(got) != 2 {
		t.Fatalf("got %d, want 2", len(got))
	}
	if got[0].GuestMount != "/" {
		t.Fatalf("expected / first (shortest), got %q", got[0].GuestMount)
	}
}
