//go:build unix

package guestfs

import "testing"

func TestParseListFilesystemsEdgeCases(t *testing.T) {
	m := parseListFilesystems("")
	if len(m) != 0 {
		t.Fatalf("empty input should give empty map, got %v", m)
	}

	m = parseListFilesystems("/dev/sda1: \n: ext4\n")
	if len(m) != 0 {
		t.Fatalf("invalid lines should be skipped, got %v", m)
	}
}

func TestDeviceBelongsToDiskEdgeCases(t *testing.T) {
	cases := []struct {
		device, disk string
		want         bool
	}{
		{"/dev/sda", "/dev/sda", true},
		{"/dev/sda1", "/dev/sda", true},
		{"/dev/sdb1", "/dev/sda", false},
		{"/dev/nvme0n1p1", "/dev/nvme0n1", true},
		{"/dev/nvme0n1", "/dev/nvme0n1", true},
		{"/dev/sda1", "/dev/sdb", false},
		{"/dev/sdaz", "/dev/sda", false},
	}
	for _, tc := range cases {
		if got := deviceBelongsToDisk(tc.device, tc.disk); got != tc.want {
			t.Errorf("deviceBelongsToDisk(%q, %q) = %v, want %v", tc.device, tc.disk, got, tc.want)
		}
	}
}

func TestDiskDeviceNameNotFound(t *testing.T) {
	b := &Backend{diskPaths: []string{"/tmp/a.img", "/tmp/b.img"}}
	_, err := b.diskDeviceName("/tmp/missing.img")
	if err == nil {
		t.Fatal("expected error for missing disk path")
	}
}
