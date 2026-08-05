package fstype

import "testing"

func TestDetectExt4(t *testing.T) {
	buf := make([]byte, 0x10100)
	buf[0x438] = 0x53
	buf[0x439] = 0xEF
	if got := detect(buf); got != "ext4" {
		t.Errorf("got %q, want ext4", got)
	}
}

func TestDetectXFS(t *testing.T) {
	buf := make([]byte, 0x10100)
	copy(buf[0:4], "XFSB")
	if got := detect(buf); got != "xfs" {
		t.Errorf("got %q, want xfs", got)
	}
}

func TestDetectBtrfs(t *testing.T) {
	buf := make([]byte, 0x10100)
	copy(buf[0x10040:0x10048], "_BHRfS_M")
	if got := detect(buf); got != "btrfs" {
		t.Errorf("got %q, want btrfs", got)
	}
}

func TestDetectNTFS(t *testing.T) {
	buf := make([]byte, 0x10100)
	copy(buf[3:7], "NTFS")
	if got := detect(buf); got != "ntfs3" {
		t.Errorf("got %q, want ntfs3", got)
	}
}

func TestDetectUnknown(t *testing.T) {
	buf := make([]byte, 0x10100)
	if got := detect(buf); got != "unknown" {
		t.Errorf("got %q, want unknown", got)
	}
}

func TestDetectSmallBuffer(t *testing.T) {
	buf := make([]byte, 100)
	if got := detect(buf); got != "unknown" {
		t.Errorf("got %q, want unknown", got)
	}
}
