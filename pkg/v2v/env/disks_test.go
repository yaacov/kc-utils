package env

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/prepare/guest/overlay"
)

func TestDiskNumber(t *testing.T) {
	cases := []struct {
		path string
		want int
	}{
		{"/dev/block/30", 30},
		{"/tmp/disks/disk1/disk.img", 1},
		{"/tmp/disks/disk12/disk.img", 12},
		{"/nodigits", 0},
	}
	for _, tc := range cases {
		if got := diskNumber(tc.path); got != tc.want {
			t.Errorf("diskNumber(%q) = %d, want %d", tc.path, got, tc.want)
		}
	}
}

func TestToOverlayDisksAndActiveDiskPaths(t *testing.T) {
	disks := []DiskInfo{{Path: "/dev/sda"}, {Path: "/dev/sdb"}}
	od := ToOverlayDisks(disks)
	if len(od) != 2 {
		t.Fatalf("ToOverlayDisks len = %d", len(od))
	}
	if od[0].BackingPath != "/dev/sda" || od[0].Path != "/dev/sda" {
		t.Errorf("overlay[0] = %+v", od[0])
	}

	od[1].Path = "/tmp/overlay-sdb"
	active := ActiveDiskPaths(od)
	if len(active) != 2 {
		t.Fatalf("ActiveDiskPaths len = %d", len(active))
	}
	if active[0].Path != "/dev/sda" || active[1].Path != "/tmp/overlay-sdb" {
		t.Errorf("active = %+v", active)
	}

	empty := ToOverlayDisks(nil)
	if len(empty) != 0 {
		t.Errorf("empty ToOverlayDisks = %v", empty)
	}
	if len(ActiveDiskPaths([]*overlay.Disk{})) != 0 {
		t.Error("empty ActiveDiskPaths should be empty")
	}
}
