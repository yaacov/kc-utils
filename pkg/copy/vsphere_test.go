package copy

import (
	"net/url"
	"testing"

	"github.com/vmware/govmomi/nfc"
	vimtypes "github.com/vmware/govmomi/vim25/types"
)

func boolPtr(v bool) *bool { return &v }

func unitPtr(v int32) *int32 { return &v }

func leaseItem(key, targetID, rawURL string, size int64, disk *bool) (vimtypes.HttpNfcLeaseDeviceUrl, nfc.FileItem) {
	u, _ := url.Parse(rawURL)
	device := vimtypes.HttpNfcLeaseDeviceUrl{
		Key:      key,
		TargetId: targetID,
		FileSize: size,
		Disk:     disk,
		Url:      rawURL,
	}
	item := nfc.NewFileItem(u, vimtypes.OvfFileItem{
		DeviceId: key,
		Path:     targetID,
		Size:     size,
	})
	return device, item
}

func flatDisk(key, controllerKey, unit int32, fileName string, parent *vimtypes.VirtualDiskFlatVer2BackingInfo) *vimtypes.VirtualDisk {
	return &vimtypes.VirtualDisk{
		VirtualDevice: vimtypes.VirtualDevice{
			Key:           key,
			ControllerKey: controllerKey,
			UnitNumber:    unitPtr(unit),
			Backing: &vimtypes.VirtualDiskFlatVer2BackingInfo{
				VirtualDeviceFileBackingInfo: vimtypes.VirtualDeviceFileBackingInfo{
					FileName: fileName,
				},
				Parent: parent,
			},
		},
	}
}

func scsiController(key int32) *vimtypes.ParaVirtualSCSIController {
	return &vimtypes.ParaVirtualSCSIController{
		VirtualSCSIController: vimtypes.VirtualSCSIController{
			VirtualController: vimtypes.VirtualController{
				VirtualDevice: vimtypes.VirtualDevice{Key: key},
			},
		},
	}
}

func TestMapDiskURLsKeyMatchDropsNonDisk(t *testing.T) {
	d0, i0 := leaseItem("2000", "disk-0.vmdk", "http://lease/disk0", 16<<30, boolPtr(true))
	d1, i1 := leaseItem("nvram", "nvram", "http://lease/nvram", 100, boolPtr(false))
	info := &nfc.LeaseInfo{
		HttpNfcLeaseInfo: vimtypes.HttpNfcLeaseInfo{
			DeviceUrl: []vimtypes.HttpNfcLeaseDeviceUrl{d0, d1},
		},
		Items: []nfc.FileItem{i0, i1},
	}
	devices := []vimtypes.BaseVirtualDevice{
		scsiController(1000),
		flatDisk(2000, 1000, 0, "[ds] vm/disk.vmdk", nil),
	}

	got := mapDiskURLs(info, devices)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1 (nvram dropped)", len(got))
	}
	if got[0].DiskPath != "[ds] vm/disk.vmdk" {
		t.Fatalf("DiskPath = %q, want backing FileName", got[0].DiskPath)
	}
	if got[0].URL != "http://lease/disk0" {
		t.Fatalf("URL = %q", got[0].URL)
	}
}

func TestMapDiskURLsPositionalFallback(t *testing.T) {
	// NFC keys do not match VirtualDisk.Key; Disk=true still selects and labels by order.
	d0, i0 := leaseItem("other-key", "disk-0.vmdk", "http://lease/a", 10, boolPtr(true))
	d1, i1 := leaseItem("other-key-2", "disk-1.vmdk", "http://lease/b", 20, boolPtr(true))
	info := &nfc.LeaseInfo{
		HttpNfcLeaseInfo: vimtypes.HttpNfcLeaseInfo{
			DeviceUrl: []vimtypes.HttpNfcLeaseDeviceUrl{d0, d1},
		},
		Items: []nfc.FileItem{i0, i1},
	}
	devices := []vimtypes.BaseVirtualDevice{
		scsiController(1000),
		flatDisk(2000, 1000, 0, "[ds] vm/a.vmdk", nil),
		flatDisk(2001, 1000, 1, "[ds] vm/b.vmdk", nil),
	}

	got := mapDiskURLs(info, devices)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].DiskPath != "[ds] vm/a.vmdk" || got[1].DiskPath != "[ds] vm/b.vmdk" {
		t.Fatalf("positional paths = %q, %q", got[0].DiskPath, got[1].DiskPath)
	}
}

func TestMapDiskURLsDeltaNormalize(t *testing.T) {
	d0, i0 := leaseItem("2000", "disk-0.vmdk", "http://lease/disk0", 16<<30, boolPtr(true))
	info := &nfc.LeaseInfo{
		HttpNfcLeaseInfo: vimtypes.HttpNfcLeaseInfo{
			DeviceUrl: []vimtypes.HttpNfcLeaseDeviceUrl{d0},
		},
		Items: []nfc.FileItem{i0},
	}
	devices := []vimtypes.BaseVirtualDevice{
		scsiController(1000),
		flatDisk(2000, 1000, 0, "[ds] vm/disk-000001.vmdk", nil),
	}

	got := mapDiskURLs(info, devices)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].DiskPath != "[ds] vm/disk.vmdk" {
		t.Fatalf("DiskPath = %q, want normalized base", got[0].DiskPath)
	}
}

func TestMapDiskURLsNoDiskFlagUsesTargetID(t *testing.T) {
	d0, i0 := leaseItem("2000", "disk-0.vmdk", "http://lease/disk0", 16<<30, nil)
	d1, i1 := leaseItem("nvram", "nvram", "http://lease/nvram", 100, nil)
	info := &nfc.LeaseInfo{
		HttpNfcLeaseInfo: vimtypes.HttpNfcLeaseInfo{
			DeviceUrl: []vimtypes.HttpNfcLeaseDeviceUrl{d0, d1},
		},
		Items: []nfc.FileItem{i0, i1},
	}
	devices := []vimtypes.BaseVirtualDevice{
		scsiController(1000),
		flatDisk(2000, 1000, 0, "[ds] vm/disk.vmdk", nil),
	}

	got := mapDiskURLs(info, devices)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1 (nvram skipped by targetId)", len(got))
	}
	if got[0].DiskPath != "[ds] vm/disk.vmdk" {
		t.Fatalf("DiskPath = %q", got[0].DiskPath)
	}
}

func TestMapDiskURLsParentBacking(t *testing.T) {
	d0, i0 := leaseItem("2000", "disk-0.vmdk", "http://lease/disk0", 16<<30, boolPtr(true))
	info := &nfc.LeaseInfo{
		HttpNfcLeaseInfo: vimtypes.HttpNfcLeaseInfo{
			DeviceUrl: []vimtypes.HttpNfcLeaseDeviceUrl{d0},
		},
		Items: []nfc.FileItem{i0},
	}
	parent := &vimtypes.VirtualDiskFlatVer2BackingInfo{
		VirtualDeviceFileBackingInfo: vimtypes.VirtualDeviceFileBackingInfo{
			FileName: "[ds] vm/disk.vmdk",
		},
	}
	devices := []vimtypes.BaseVirtualDevice{
		scsiController(1000),
		flatDisk(2000, 1000, 0, "[ds] vm/disk-000001.vmdk", parent),
	}

	got := mapDiskURLs(info, devices)
	if len(got) != 1 || got[0].DiskPath != "[ds] vm/disk.vmdk" {
		t.Fatalf("got %+v, want parent base path", got)
	}
}
