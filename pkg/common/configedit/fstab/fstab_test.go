package fstab

import (
	"strings"
	"testing"
)

const testFstab = `# /etc/fstab
/dev/sda1	/	ext4	defaults	0	1
/dev/sda2	/home	ext4	defaults	0	2
UUID=abc-123	/boot	xfs	defaults	0	0
`

func TestParse(t *testing.T) {
	f := Parse(testFstab)
	devEntries := f.DeviceEntries()
	if len(devEntries) != 3 {
		t.Fatalf("got %d device entries, want 3", len(devEntries))
	}
	if devEntries[0].Device != "/dev/sda1" {
		t.Errorf("first device = %q, want /dev/sda1", devEntries[0].Device)
	}
	if devEntries[0].MountPoint != "/" {
		t.Errorf("first mount = %q, want /", devEntries[0].MountPoint)
	}
}

func TestRemapDevice(t *testing.T) {
	f := Parse(testFstab)
	f.RemapDevice("/dev/sda", "/dev/vda")
	devEntries := f.DeviceEntries()
	if devEntries[0].Device != "/dev/vda1" {
		t.Errorf("got %q, want /dev/vda1", devEntries[0].Device)
	}
	if !strings.HasPrefix(devEntries[2].Device, "UUID=") {
		t.Error("UUID entry should not be remapped")
	}
}

func TestString(t *testing.T) {
	f := Parse(testFstab)
	out := f.String()
	if !strings.Contains(out, "/dev/sda1") {
		t.Error("output should contain device")
	}
	if !strings.Contains(out, "# /etc/fstab") {
		t.Error("output should preserve comments")
	}
}

func TestRemapAllFields(t *testing.T) {
	content := "luks-root\t/dev/sda2\tnone\tluks\n"
	f := Parse(content)
	f.RemapAllFields("/dev/sda", "/dev/vda")
	entries := f.DeviceEntries()
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].MountPoint != "/dev/vda2" {
		t.Errorf("MountPoint = %q, want /dev/vda2", entries[0].MountPoint)
	}
	if entries[0].Device != "luks-root" {
		t.Errorf("Device should remain luks-root, got %q", entries[0].Device)
	}
}

func TestRemapAllFieldsSkipsComments(t *testing.T) {
	content := "# /dev/sda1 is a comment\n/dev/sda1\t/\text4\tdefaults\n"
	f := Parse(content)
	f.RemapAllFields("/dev/sda", "/dev/vda")
	out := f.String()
	if !strings.Contains(out, "# /dev/sda1 is a comment") {
		t.Error("comment should not be remapped")
	}
	if !strings.Contains(out, "/dev/vda1") {
		t.Error("device should be remapped")
	}
}

func TestRemapAllFieldsDeviceColumn(t *testing.T) {
	content := "/dev/sda1\t/boot\text4\tdefaults\t0\t1\n"
	f := Parse(content)
	f.RemapAllFields("/dev/sda", "/dev/vda")
	entries := f.DeviceEntries()
	if entries[0].Device != "/dev/vda1" {
		t.Errorf("Device = %q, want /dev/vda1", entries[0].Device)
	}
}

func TestParseEmptyContent(t *testing.T) {
	f := Parse("")
	entries := f.DeviceEntries()
	if len(entries) != 0 {
		t.Errorf("got %d entries for empty content, want 0", len(entries))
	}
}

func TestStringDefaultsForMissing(t *testing.T) {
	f := &File{
		Entries: []Entry{
			{Device: "/dev/sda1", MountPoint: "/"},
		},
	}
	out := f.String()
	if !strings.Contains(out, "defaults") {
		t.Error("should fill in defaults for missing options")
	}
}
