package gptesp

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestDetectUEFI(t *testing.T) {
	d := &GPTESPDetector{}
	disks := []types.DiskInfo{{
		Partitions: []types.PartitionInfo{
			{Index: 1, DevicePath: "/dev/sda1", FSType: "vfat", SizeBytes: 512 * 1024 * 1024},
			{Index: 2, DevicePath: "/dev/sda2", FSType: "ext4"},
		},
	}}
	fw, err := d.Detect(disks)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}
	if fw.Type != string(types.FirmwareUEFI) {
		t.Errorf("Type = %q, want %q", fw.Type, types.FirmwareUEFI)
	}
	if len(fw.ESPDevices) != 1 || fw.ESPDevices[0] != "/dev/sda1" {
		t.Errorf("ESPDevices = %v, want [/dev/sda1]", fw.ESPDevices)
	}
}

func TestDetectBIOSNoVfat(t *testing.T) {
	d := &GPTESPDetector{}
	disks := []types.DiskInfo{{
		Partitions: []types.PartitionInfo{
			{Index: 1, DevicePath: "/dev/sda1", FSType: "ext4"},
		},
	}}
	fw, err := d.Detect(disks)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}
	if fw.Type != string(types.FirmwareBIOS) {
		t.Errorf("Type = %q, want %q", fw.Type, types.FirmwareBIOS)
	}
}

func TestDetectEmptyDisks(t *testing.T) {
	d := &GPTESPDetector{}
	fw, err := d.Detect(nil)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}
	if fw.Type != string(types.FirmwareBIOS) {
		t.Errorf("Type = %q, want %q", fw.Type, types.FirmwareBIOS)
	}
}

func TestDetectUEFISecondDisk(t *testing.T) {
	d := &GPTESPDetector{}
	disks := []types.DiskInfo{
		{Partitions: []types.PartitionInfo{{Index: 1, FSType: "ext4"}}},
		{Partitions: []types.PartitionInfo{{Index: 1, DevicePath: "/dev/sdb1", FSType: "vfat", SizeBytes: 256 * 1024 * 1024}}},
	}
	fw, err := d.Detect(disks)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}
	if fw.Type != string(types.FirmwareUEFI) {
		t.Errorf("Type = %q, want %q", fw.Type, types.FirmwareUEFI)
	}
	if len(fw.ESPDevices) != 1 || fw.ESPDevices[0] != "/dev/sdb1" {
		t.Errorf("ESPDevices = %v, want [/dev/sdb1]", fw.ESPDevices)
	}
}

func TestDetectBIOSForLargeDataVFAT(t *testing.T) {
	d := &GPTESPDetector{}
	disks := []types.DiskInfo{{
		Partitions: []types.PartitionInfo{
			{Index: 3, DevicePath: "/dev/sda3", FSType: "vfat", SizeBytes: 4 * 1024 * 1024 * 1024},
		},
	}}
	fw, err := d.Detect(disks)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}
	if fw.Type != string(types.FirmwareBIOS) {
		t.Errorf("Type = %q, want %q", fw.Type, types.FirmwareBIOS)
	}
}

func TestDetectUEFIFromBootEFIMount(t *testing.T) {
	d := &GPTESPDetector{}
	disks := []types.DiskInfo{{
		Partitions: []types.PartitionInfo{
			{Index: 3, DevicePath: "/dev/sda3", FSType: "vfat", MountPoint: "/boot/efi", SizeBytes: 4 * 1024 * 1024 * 1024},
		},
	}}
	fw, err := d.Detect(disks)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}
	if fw.Type != string(types.FirmwareUEFI) {
		t.Errorf("Type = %q, want %q", fw.Type, types.FirmwareUEFI)
	}
}
