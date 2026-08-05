package env

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestBuildPrepareInput(t *testing.T) {
	cfg := &Config{
		Workdir:                      "/work",
		HostName:                     "guest1",
		DynamicScriptsDir:            "/scripts",
		VsphereVmwareDriverRemoval:   true,
		WindowsRegistryNetworkConfig: true,
		MultipleIPsPerNic:            true,
		WaitForGuestReboot:           true,
		StaticIPs:                    "52:54:00:aa:bb:cc:ip:192.168.1.10,192.168.1.1,24,8.8.8.8",
	}
	disks := []DiskInfo{{Path: "/dev/sda"}, {Path: "/dev/sdb"}}
	source := types.SourceSpec{Name: "vm1", Type: "vsphere"}

	in, err := BuildPrepareInput(cfg, disks, source)
	if err != nil {
		t.Fatalf("BuildPrepareInput: %v", err)
	}
	if len(in.Disks) != 2 || in.Disks[0].Path != "/dev/sda" || in.Disks[0].Format != "raw" {
		t.Errorf("disks = %+v", in.Disks)
	}
	if in.Source.Name != "vm1" || in.Source.Type != "vsphere" {
		t.Errorf("source = %+v", in.Source)
	}
	if in.Options.Root != "first" {
		t.Errorf("Root = %q, want first", in.Options.Root)
	}
	if in.Options.TmpDir != "/work" || in.Options.Hostname != "guest1" {
		t.Errorf("options = %+v", in.Options)
	}
	if len(in.Options.StaticIPs) != 1 || in.Options.StaticIPs[0].IP != "192.168.1.10" {
		t.Errorf("StaticIPs = %+v", in.Options.StaticIPs)
	}
	if !in.Options.VMwareDriverRemoval || !in.Options.WindowsRegistryNetwork ||
		!in.Options.MultipleIPsPerNic || !in.Options.WaitForGuestReboot {
		t.Errorf("bool options = %+v", in.Options)
	}
	if in.Options.DynamicScriptsDir != "/scripts" {
		t.Errorf("paths = %+v", in.Options)
	}
}

func TestBuildPrepareInputRootOverride(t *testing.T) {
	cfg := &Config{RootDisk: "/dev/sdb"}
	in, err := BuildPrepareInput(cfg, nil, types.SourceSpec{})
	if err != nil {
		t.Fatalf("BuildPrepareInput: %v", err)
	}
	if in.Options.Root != "/dev/sdb" {
		t.Errorf("Root = %q", in.Options.Root)
	}
}

func TestBuildPrepareInputInvalidStaticIPs(t *testing.T) {
	cfg := &Config{StaticIPs: "not-a-valid-segment"}
	_, err := BuildPrepareInput(cfg, nil, types.SourceSpec{})
	if err == nil {
		t.Fatal("expected error for invalid static IPs")
	}
}
