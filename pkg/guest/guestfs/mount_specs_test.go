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
	got, err := b.effectiveMountSpecs()
	if err != nil {
		t.Fatalf("effectiveMountSpecs: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d, want 2", len(got))
	}
	if got[0].GuestMount != "/" {
		t.Fatalf("expected / first (shortest), got %q", got[0].GuestMount)
	}
}

func TestPreferredRoot(t *testing.T) {
	if got := preferredRoot(nil); got != "" {
		t.Fatalf("empty specs: %q", got)
	}
	specs := []gfsMountSpec{
		{Device: "/dev/sda1", GuestMount: "/boot"},
		{Device: "/dev/sda2", GuestMount: "/"},
	}
	if got := preferredRoot(specs); got != "/dev/sda2" {
		t.Fatalf("preferredRoot=%q want /dev/sda2", got)
	}
}

func TestSelectInspectRoot(t *testing.T) {
	roots := []string{"/dev/sda2", "/dev/sda3"}

	got, err := selectInspectRoot(roots, "")
	if err != nil || got != "/dev/sda2" {
		t.Fatalf("empty preferred: got=%q err=%v want first", got, err)
	}

	got, err = selectInspectRoot(roots, "/dev/sda3")
	if err != nil || got != "/dev/sda3" {
		t.Fatalf("match non-first: got=%q err=%v", got, err)
	}

	got, err = selectInspectRoot(roots, "/dev/mapper/rhel-root")
	if err == nil || got != "" {
		t.Fatalf("mismatch should error: got=%q err=%v", got, err)
	}

	got, err = selectInspectRoot(nil, "/dev/sda2")
	if err != nil || got != "" {
		t.Fatalf("no roots: got=%q err=%v", got, err)
	}
}

func TestSelectInspectRootLVMAlias(t *testing.T) {
	roots := []string{"/dev/rhel/root", "/dev/sda2"}
	got, err := selectInspectRoot(roots, "/dev/mapper/rhel-root")
	if err != nil || got != "/dev/rhel/root" {
		t.Fatalf("LVM alias: got=%q err=%v", got, err)
	}
}

func TestDeviceEqual(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"/dev/sda1", "/dev/sda1", true},
		{"/dev/sda1", "/dev/sda2", false},
		{"/dev/rhel/root", "/dev/mapper/rhel-root", true},
		{"/dev/mapper/rhel-root", "/dev/rhel/root", true},
		{"/dev/my-vg/my-lv", "/dev/mapper/my--vg-my--lv", true},
		{"/dev/rhel/root", "/dev/mapper/other-root", false},
		{"/dev/sda1", "/dev/mapper/sda1", true}, // leaf basename last resort
		{"/dev/rhel/root", "/dev/fedora/root", false},
		{"", "/dev/sda1", false},
	}
	for _, tc := range cases {
		if got := deviceEqual(tc.a, tc.b); got != tc.want {
			t.Errorf("deviceEqual(%q, %q)=%v want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestMergeMountSpecsPathConflictKeepsBase(t *testing.T) {
	base := []gfsMountSpec{
		{Device: "/dev/sda2", GuestMount: "/"},
		{Device: "/dev/sda3", GuestMount: "/boot"},
	}
	extra := []gfsMountSpec{
		{Device: "/dev/sda2", GuestMount: "/"},
		{Device: "/dev/sda1", GuestMount: "/boot"}, // different device, same path
		{Device: "/dev/sda4", GuestMount: "/boot/efi"},
	}
	got := mergeMountSpecs(base, extra)
	if len(got) != 3 {
		t.Fatalf("got %d want 3: %#v", len(got), got)
	}
	for _, s := range got {
		if s.GuestMount == "/boot" && s.Device != "/dev/sda3" {
			t.Fatalf("path conflict should keep prepare device, got %#v", s)
		}
	}
}

func TestParseInspectOSRoots(t *testing.T) {
	roots := parseInspectOSRoots("/dev/sda2\n/dev/sda3\nnot-a-root\n")
	if len(roots) != 2 || roots[0] != "/dev/sda2" || roots[1] != "/dev/sda3" {
		t.Fatalf("%v", roots)
	}
}
