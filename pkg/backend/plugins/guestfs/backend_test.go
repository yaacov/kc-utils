//go:build linux

package guestfs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestSortedMountSpecs(t *testing.T) {
	specs := []gfsMountSpec{
		{Device: "/dev/sda2", GuestMount: "/boot/efi"},
		{Device: "/dev/sda3", GuestMount: "/"},
		{Device: "/dev/sda1", GuestMount: "/boot"},
	}
	got := sortedMountSpecs(specs)
	if got[0].GuestMount != "/" || got[1].GuestMount != "/boot" || got[2].GuestMount != "/boot/efi" {
		t.Fatalf("unexpected order: %#v", got)
	}
}

func TestClearDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := clearDir(dir); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty dir, got %v", entries)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatal(err)
	}
}

func TestMountSpecsFromDiskInfos(t *testing.T) {
	disks := []types.DiskInfo{{
		Partitions: []types.PartitionInfo{
			{DevicePath: "/dev/sda1", FSType: "ext4", MountPoint: "/"},
			{DevicePath: "/dev/sda2", FSType: "vfat", MountPoint: "/boot/efi"},
			{DevicePath: "/dev/sda3", FSType: "swap"},
		},
	}}
	specs := mountSpecsFromDiskInfos(disks)
	if len(specs) != 2 {
		t.Fatalf("got %d specs, want 2: %#v", len(specs), specs)
	}
}

func TestPruneEmptyWindowsProbeDirs(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "Windows", "System32", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	pruneEmptyWindowsProbeDirs(dir)
	if _, err := os.Stat(filepath.Join(dir, "Windows")); !os.IsNotExist(err) {
		t.Fatalf("expected Windows tree removed, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "etc")); !os.IsNotExist(err) {
		t.Fatalf("expected empty etc removed, err=%v", err)
	}

	dir2 := t.TempDir()
	hive := filepath.Join(dir2, "Windows", "System32", "config", "SYSTEM")
	if err := os.MkdirAll(filepath.Dir(hive), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hive, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	pruneEmptyWindowsProbeDirs(dir2)
	if _, err := os.Stat(hive); err != nil {
		t.Fatalf("expected hive kept: %v", err)
	}
}

func TestQuoteGuestfish(t *testing.T) {
	if quoteGuestfish(`/Program Files/VMware`) != `"/Program Files/VMware"` {
		t.Fatalf("quoteGuestfish: %q", quoteGuestfish(`/Program Files/VMware`))
	}
	if quoteGuestfish(`/etc/fstab`) != `/etc/fstab` {
		t.Fatalf("quoteGuestfish plain: %q", quoteGuestfish(`/etc/fstab`))
	}
}

func TestMountScriptPrefixNoRun(t *testing.T) {
	script := mountScriptPrefix([]gfsMountSpec{
		{Device: "/dev/sda2", GuestMount: "/boot"},
		{Device: "/dev/sda1", GuestMount: "/"},
	})
	if script[:len("-umount-all\n")] != "-umount-all\n" {
		t.Fatalf("prefix: %q", script)
	}
	if !containsInOrder(script, "-mount /dev/sda1 /", "-mount /dev/sda2 /boot") {
		t.Fatalf("mount order: %s", script)
	}
}

func TestDeviceBelongsToDisk(t *testing.T) {
	cases := []struct {
		device, disk string
		want         bool
	}{
		{"/dev/sda1", "/dev/sda", true},
		{"/dev/sda", "/dev/sda", true},
		{"/dev/sdb1", "/dev/sda", false},
		{"/dev/nvme0n1p1", "/dev/nvme0n1", true},
		{"/dev/sda1", "/dev/sdb", false},
	}
	for _, tc := range cases {
		if got := deviceBelongsToDisk(tc.device, tc.disk); got != tc.want {
			t.Fatalf("deviceBelongsToDisk(%q,%q)=%v want %v", tc.device, tc.disk, got, tc.want)
		}
	}
}

func TestParseListFilesystems(t *testing.T) {
	m := parseListFilesystems("/dev/sda1: ext4\n/dev/sda2: vfat\n\nbadline\n")
	if m["/dev/sda1"] != "ext4" || m["/dev/sda2"] != "vfat" {
		t.Fatalf("%v", m)
	}
}

func TestMergeMountSpecs(t *testing.T) {
	base := []gfsMountSpec{
		{Device: "/dev/sda2", GuestMount: "/"},
	}
	extra := []gfsMountSpec{
		{Device: "/dev/sda2", GuestMount: "/"},
		{Device: "/dev/sda1", GuestMount: "/boot"},
		{Device: "/dev/sda3", GuestMount: "/boot/efi"},
	}
	got := mergeMountSpecs(base, extra)
	if len(got) != 3 {
		t.Fatalf("expected 3 specs (/ + /boot + /boot/efi), got %d: %#v", len(got), got)
	}
	mounts := map[string]bool{}
	for _, s := range got {
		mounts[s.GuestMount] = true
	}
	if !mounts["/boot"] || !mounts["/boot/efi"] {
		t.Fatalf("missing /boot or /boot/efi in merged specs: %#v", got)
	}
}

func TestMergeMountSpecsNoDuplicates(t *testing.T) {
	base := []gfsMountSpec{
		{Device: "/dev/sda2", GuestMount: "/"},
		{Device: "/dev/sda1", GuestMount: "/boot"},
	}
	extra := []gfsMountSpec{
		{Device: "/dev/sda1", GuestMount: "/boot"},
	}
	got := mergeMountSpecs(base, extra)
	if len(got) != 2 {
		t.Fatalf("expected 2 specs (no duplicate /boot), got %d: %#v", len(got), got)
	}
}

func TestRunCommandQuotesArgs(t *testing.T) {
	cmd := []string{"dracut", "--force", "--add-drivers", "virtio virtio_blk virtio_scsi"}
	expected := `sh "dracut --force --add-drivers 'virtio virtio_blk virtio_scsi' 2>&1"` + "\n"
	if got := shCommandScript(cmd); got != expected {
		t.Fatalf("sh command quoting:\ngot:  %q\nwant: %q", got, expected)
	}
}

func containsInOrder(s, a, b string) bool {
	i := strings.Index(s, a)
	j := strings.Index(s, b)
	return i >= 0 && j > i
}

func TestFscheckCommand(t *testing.T) {
	tests := []struct {
		fstype string
		cmd    string
		ok     bool
	}{
		{"ext4", "e2fsck-f", true},
		{"ext3", "e2fsck-f", true},
		{"ext2", "e2fsck-f", true},
		{"xfs", "xfs-repair", true},
		{"ntfs", "ntfsfix", true},
		{"ntfs3", "ntfsfix", true},
		{"NTFS", "ntfsfix", true},
		{"vfat", "", false},
		{"btrfs", "", false},
		{"unknown", "", false},
	}
	for _, tt := range tests {
		cmd, ok := fscheckCommand(tt.fstype)
		if ok != tt.ok || cmd != tt.cmd {
			t.Errorf("fscheckCommand(%q) = (%q, %v), want (%q, %v)", tt.fstype, cmd, ok, tt.cmd, tt.ok)
		}
	}
}
