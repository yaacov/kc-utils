package target

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestTarget(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", "bios"},
		{"bios", "bios"},
		{"uefi", "uefi"},
	}
	for _, tc := range cases {
		if got := Target(tc.in); got != tc.want {
			t.Errorf("Target(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestBuses(t *testing.T) {
	disks := []types.DiskInfo{{Path: "/dev/sda"}, {Path: "/dev/sdb"}}

	virtio := Buses(disks, "virtio")
	if len(virtio.VirtioBlk) != 2 || len(virtio.SCSI) != 0 || len(virtio.IDE) != 0 {
		t.Fatalf("virtio buses = %+v", virtio)
	}
	if virtio.VirtioBlk[0].Index != 0 || virtio.VirtioBlk[0].SourceDisk != 0 {
		t.Errorf("virtio slot0 = %+v", virtio.VirtioBlk[0])
	}
	if virtio.VirtioBlk[1].Index != 1 || virtio.VirtioBlk[1].SourceDisk != 1 {
		t.Errorf("virtio slot1 = %+v", virtio.VirtioBlk[1])
	}

	scsi := Buses(disks, "scsi")
	if len(scsi.SCSI) != 2 || len(scsi.VirtioBlk) != 0 || len(scsi.IDE) != 0 {
		t.Fatalf("scsi buses = %+v", scsi)
	}

	ide := Buses(disks, "ide")
	if len(ide.IDE) != 2 || len(ide.VirtioBlk) != 0 || len(ide.SCSI) != 0 {
		t.Fatalf("ide buses = %+v", ide)
	}

	empty := Buses(nil, "virtio")
	if len(empty.VirtioBlk) != 0 || len(empty.SCSI) != 0 || len(empty.IDE) != 0 {
		t.Fatalf("empty disks = %+v", empty)
	}
}

func TestNICs(t *testing.T) {
	defaults := NICs(nil, "virtio")
	if len(defaults) != 1 {
		t.Fatalf("empty source NICs len = %d", len(defaults))
	}
	if defaults[0].MAC != "00:00:00:00:00:00" || defaults[0].Model != "virtio" {
		t.Errorf("default NIC = %+v", defaults[0])
	}

	src := []types.NICSpec{
		{MAC: "52:54:00:aa:bb:cc", Network: "net1"},
		{MAC: "52:54:00:dd:ee:ff", Network: "net2"},
	}
	nics := NICs(src, "e1000")
	if len(nics) != 2 {
		t.Fatalf("NICs len = %d", len(nics))
	}
	if nics[0].MAC != "52:54:00:aa:bb:cc" || nics[0].Model != "e1000" || nics[0].Network != "net1" {
		t.Errorf("nic0 = %+v", nics[0])
	}
	if nics[1].MAC != "52:54:00:dd:ee:ff" || nics[1].Model != "e1000" || nics[1].Network != "net2" {
		t.Errorf("nic1 = %+v", nics[1])
	}
}
