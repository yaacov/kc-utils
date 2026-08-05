package vsphere

import (
	"testing"

	vimtypes "github.com/vmware/govmomi/vim25/types"
)

func TestSDKURL(t *testing.T) {
	sdk, insecure, err := sdkURL("vpx://user@vcenter.example.com/Datacenter/Cluster/host?no_verify=1")
	if err != nil {
		t.Fatal(err)
	}
	if sdk.Scheme != "https" || sdk.Host != "vcenter.example.com" || sdk.Path != "/sdk" {
		t.Fatalf("sdk URL = %s", sdk)
	}
	if !insecure {
		t.Fatal("expected insecure")
	}
}

func TestSDKURLWithPort(t *testing.T) {
	sdk, _, err := sdkURL("vpx://user@vcenter.example.com:8443/path")
	if err != nil {
		t.Fatal(err)
	}
	if sdk.Host != "vcenter.example.com:8443" {
		t.Fatalf("host = %q", sdk.Host)
	}
}

func TestDatacenterName(t *testing.T) {
	if got := datacenterName("vpx://u@vc/Datacenter/Cluster/host"); got != "Datacenter" {
		t.Fatalf("got %q", got)
	}
}

func TestTrimDeltaSuffix(t *testing.T) {
	in := "[ds] vm/vm-000002.vmdk"
	want := "[ds] vm/vm.vmdk"
	if got := trimDeltaSuffix(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got := trimDeltaSuffix("[ds] vm/vm.vmdk"); got != "[ds] vm/vm.vmdk" {
		t.Fatalf("got %q", got)
	}
}

func TestDisksFromDevicesOrder(t *testing.T) {
	// Minimal device graph: SCSI ctrl key=1000, disk on unit 0; SATA ctrl key=15000, disk unit 0
	scsiKey := int32(1000)
	sataKey := int32(15000)
	unit0 := int32(0)

	devices := []vimtypes.BaseVirtualDevice{
		&vimtypes.ParaVirtualSCSIController{
			VirtualSCSIController: vimtypes.VirtualSCSIController{
				VirtualController: vimtypes.VirtualController{
					VirtualDevice: vimtypes.VirtualDevice{Key: scsiKey},
				},
			},
		},
		&vimtypes.VirtualSATAController{
			VirtualController: vimtypes.VirtualController{
				VirtualDevice: vimtypes.VirtualDevice{Key: sataKey},
			},
		},
		&vimtypes.VirtualDisk{
			VirtualDevice: vimtypes.VirtualDevice{
				Key:           2000,
				ControllerKey: sataKey,
				UnitNumber:    &unit0,
				Backing: &vimtypes.VirtualDiskFlatVer2BackingInfo{
					VirtualDeviceFileBackingInfo: vimtypes.VirtualDeviceFileBackingInfo{
						FileName: "[ds] vm/sata.vmdk",
					},
				},
			},
		},
		&vimtypes.VirtualDisk{
			VirtualDevice: vimtypes.VirtualDevice{
				Key:           2001,
				ControllerKey: scsiKey,
				UnitNumber:    &unit0,
				Backing: &vimtypes.VirtualDiskFlatVer2BackingInfo{
					VirtualDeviceFileBackingInfo: vimtypes.VirtualDeviceFileBackingInfo{
						FileName: "[ds] vm/scsi.vmdk",
					},
				},
			},
		},
	}

	paths := disksFromDevices(devices)
	if len(paths) != 2 {
		t.Fatalf("got %v", paths)
	}
	// libvirt order: SCSI before SATA
	if paths[0] != "[ds] vm/scsi.vmdk" || paths[1] != "[ds] vm/sata.vmdk" {
		t.Fatalf("order = %v", paths)
	}
}
