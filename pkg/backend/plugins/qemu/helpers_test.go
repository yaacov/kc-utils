//go:build unix

package qemu

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestApplianceMountPath(t *testing.T) {
	cases := []struct {
		mountRoot string
		host      string
		want      string
	}{
		{"/mnt/root", "/mnt/root", "/mnt/guest"}, // root itself
		{"/mnt/root", "/mnt/root/boot", "/mnt/guest/boot"},
		{"/mnt/root", "/mnt/root/boot/efi", "/mnt/guest/boot/efi"},
		{"", "/anything", "/mnt/guest"}, // empty mountRoot
	}
	for _, c := range cases {
		if got := applianceMountPath(c.mountRoot, c.host); got != c.want {
			t.Errorf("applianceMountPath(%q, %q) = %q, want %q", c.mountRoot, c.host, got, c.want)
		}
	}
}

func TestGuestToAppliance(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/etc/fstab", "/mnt/guest/etc/fstab"},
		{"/", "/mnt/guest"},
		// Traversal must stay confined under the mount root.
		{"../../etc/passwd", "/mnt/guest/etc/passwd"},
		{"/../../../etc/shadow", "/mnt/guest/etc/shadow"},
		{"/a/b/../../../../c", "/mnt/guest/c"},
	}
	for _, c := range cases {
		if got := guestToAppliance(c.in); got != c.want {
			t.Errorf("guestToAppliance(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDiskDevice(t *testing.T) {
	cases := []struct {
		i    int
		want string
	}{
		{0, "/dev/vda"},
		{1, "/dev/vdb"},
		{25, "/dev/vdz"},
		{26, "/dev/vdaa"},
		{27, "/dev/vdab"},
		{51, "/dev/vdaz"},
		{52, "/dev/vdba"},
		{701, "/dev/vdzz"},
		{702, "/dev/vdaaa"},
	}
	for _, c := range cases {
		if got := diskDevice(c.i); got != c.want {
			t.Errorf("diskDevice(%d) = %q, want %q", c.i, got, c.want)
		}
	}
}

func TestEscapeQEMUValue(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/tmp/plain.img", "/tmp/plain.img"},
		{"/tmp/a,b.img", "/tmp/a,,b.img"},
		{"a,b,c", "a,,b,,c"},
	}
	for _, c := range cases {
		if got := escapeQEMUValue(c.in); got != c.want {
			t.Errorf("escapeQEMUValue(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLsblkTopSize(t *testing.T) {
	out := []byte(`{"blockdevices":[{"name":"vdb","path":"/dev/vdb","type":"disk","fstype":"","size":4096}]}`)
	if got := lsblkTopSize(out); got != 4096 {
		t.Errorf("lsblkTopSize = %d, want 4096", got)
	}
	if got := lsblkTopSize([]byte(`{"blockdevices":[]}`)); got != 0 {
		t.Errorf("lsblkTopSize(empty) = %d, want 0", got)
	}
	if got := lsblkTopSize([]byte("not json")); got != 0 {
		t.Errorf("lsblkTopSize(bad) = %d, want 0", got)
	}
}

func TestHostMountFromGuest(t *testing.T) {
	if got := hostMountFromGuest("/mnt/root", "/"); got != "/mnt/root" {
		t.Errorf("hostMountFromGuest root = %q", got)
	}
	if got := hostMountFromGuest("/mnt/root", ""); got != "/mnt/root" {
		t.Errorf("hostMountFromGuest empty = %q", got)
	}
	if got := hostMountFromGuest("/mnt/root", "/boot"); got != "/mnt/root/boot" {
		t.Errorf("hostMountFromGuest /boot = %q", got)
	}
	if got := hostMountFromGuest("/mnt/root", "/../../proc"); got != "/mnt/root/proc" {
		t.Errorf("hostMountFromGuest traversal = %q, want confined under mountRoot", got)
	}
	if got := hostMountFromGuest("/mnt/root", "../../etc"); got != "/mnt/root/etc" {
		t.Errorf("hostMountFromGuest relative traversal = %q", got)
	}
}

func TestDetectImageFormat(t *testing.T) {
	dir := t.TempDir()

	qcow := filepath.Join(dir, "d.qcow2")
	if err := os.WriteFile(qcow, []byte{'Q', 'F', 'I', 0xfb, 0, 0}, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := detectImageFormat(qcow); got != "qcow2" {
		t.Errorf("detectImageFormat(qcow2 magic) = %q", got)
	}

	raw := filepath.Join(dir, "d.img")
	if err := os.WriteFile(raw, []byte{0, 1, 2, 3, 4}, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := detectImageFormat(raw); got != "raw" {
		t.Errorf("detectImageFormat(raw) = %q", got)
	}

	// Missing file falls back to raw rather than erroring.
	if got := detectImageFormat(filepath.Join(dir, "nope")); got != "raw" {
		t.Errorf("detectImageFormat(missing) = %q", got)
	}
}

func TestToDriveSpecs(t *testing.T) {
	dir := t.TempDir()
	qcow := filepath.Join(dir, "d.qcow2")
	if err := os.WriteFile(qcow, []byte{'Q', 'F', 'I', 0xfb}, 0o644); err != nil {
		t.Fatal(err)
	}

	disks := []types.DiskSpec{
		{Path: "/explicit.img", Format: "raw"}, // explicit format preserved
		{Path: qcow},                           // format sniffed -> qcow2
	}
	got := toDriveSpecs(disks)
	want := []driveSpec{
		{Path: "/explicit.img", Format: "raw"},
		{Path: qcow, Format: "qcow2"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("toDriveSpecs = %+v, want %+v", got, want)
	}
}

func TestEnvPositiveInt(t *testing.T) {
	t.Setenv("KC_TEST_A", "")
	t.Setenv("KC_TEST_B", "0")
	t.Setenv("KC_TEST_C", "6")
	if got := envPositiveInt(9, "KC_TEST_A", "KC_TEST_B", "KC_TEST_C"); got != 6 {
		t.Errorf("envPositiveInt = %d, want 6 (skips empty and non-positive)", got)
	}
	if got := envPositiveInt(9, "KC_TEST_A", "KC_TEST_B"); got != 9 {
		t.Errorf("envPositiveInt fallback = %d, want 9", got)
	}
}

func TestFsckArgv(t *testing.T) {
	cases := []struct {
		fs   string
		want []string
	}{
		{"ext4", []string{"e2fsck", "-f", "-y", "/dev/vda1"}},
		{"EXT3", []string{"e2fsck", "-f", "-y", "/dev/vda1"}}, // case-insensitive
		{"xfs", []string{"xfs_repair", "/dev/vda1"}},
		{"btrfs", []string{"btrfs", "check", "/dev/vda1"}},
		{"ntfs", []string{"ntfsfix", "-d", "/dev/vda1"}},
		{"swap", nil}, // unknown/unsupported -> skipped
		{"", nil},
	}
	for _, c := range cases {
		got := fsckArgv("/dev/vda1", c.fs)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("fsckArgv(%q) = %v, want %v", c.fs, got, c.want)
		}
	}
}

func TestParseLsblkPartitions(t *testing.T) {
	out := []byte(`{"blockdevices":[{"name":"vda","path":"/dev/vda","type":"disk","fstype":"","size":100,"children":[
		{"name":"vda1","path":"/dev/vda1","type":"part","fstype":"vfat","size":50},
		{"name":"vda2","path":"/dev/vda2","type":"part","fstype":"ext4","size":50}
	]}]}`)
	parts, err := parseLsblkPartitions(out)
	if err != nil {
		t.Fatalf("parseLsblkPartitions: %v", err)
	}
	if len(parts) != 2 {
		t.Fatalf("got %d partitions, want 2", len(parts))
	}
	if parts[0].Index != 1 || parts[0].DevicePath != "/dev/vda1" || parts[0].FSType != "vfat" {
		t.Errorf("part[0] = %+v", parts[0])
	}
	if parts[1].Index != 2 || parts[1].DevicePath != "/dev/vda2" || parts[1].FSType != "ext4" {
		t.Errorf("part[1] = %+v", parts[1])
	}
}

func TestParseLsblkWholeDisk(t *testing.T) {
	// No partition table: whole-disk filesystem reported as a single partition.
	out := []byte(`{"blockdevices":[{"name":"vdb","path":"/dev/vdb","type":"disk","fstype":"xfs","size":200}]}`)
	parts, err := parseLsblkPartitions(out)
	if err != nil {
		t.Fatalf("parseLsblkPartitions: %v", err)
	}
	if len(parts) != 1 || parts[0].DevicePath != "/dev/vdb" || parts[0].FSType != "xfs" {
		t.Fatalf("whole-disk parts = %+v", parts)
	}
}

func TestParseLsblkEmpty(t *testing.T) {
	parts, err := parseLsblkPartitions([]byte(`{"blockdevices":[]}`))
	if err != nil {
		t.Fatalf("parseLsblkPartitions: %v", err)
	}
	if parts != nil {
		t.Errorf("expected nil parts, got %+v", parts)
	}
}

func TestParseLVPaths(t *testing.T) {
	out := []byte("  /dev/vg0/root  \n\n  /dev/vg0/swap\n")
	got := parseLVPaths(out)
	want := []string{"/dev/vg0/root", "/dev/vg0/swap"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseLVPaths = %v, want %v", got, want)
	}
	if parseLVPaths([]byte("  \n\n")) != nil {
		t.Errorf("expected nil for blank output")
	}
}

func TestOrderMountPlans(t *testing.T) {
	infos := []types.DiskInfo{
		{Partitions: []types.PartitionInfo{
			{DevicePath: "/dev/vda2", MountPoint: "/boot", FSType: "ext4"},
			{DevicePath: "/dev/vda1", MountPoint: "/", FSType: "ext4"},
			{DevicePath: "/dev/vda3", MountPoint: "/boot/efi", FSType: "vfat"},
			{DevicePath: "/dev/vda4", MountPoint: "", FSType: "ext4"},     // no mountpoint -> skip
			{DevicePath: "/dev/vda5", MountPoint: "swap", FSType: "swap"}, // swap -> skip
		}},
	}
	plans := orderMountPlans(infos)
	// Expect shortest-first: "/", "/boot", "/boot/efi".
	want := []mountPlan{
		{device: "/dev/vda1", mount: "/"},
		{device: "/dev/vda2", mount: "/boot"},
		{device: "/dev/vda3", mount: "/boot/efi"},
	}
	if !reflect.DeepEqual(plans, want) {
		t.Errorf("orderMountPlans = %+v, want %+v", plans, want)
	}
}

func TestMountEntriesFromDiskInfos(t *testing.T) {
	infos := []types.DiskInfo{
		{Partitions: []types.PartitionInfo{
			{DevicePath: "/dev/vda3", MountPoint: "/", FSType: "xfs"},
			{DevicePath: "/dev/vda4", MountPoint: "/boot", FSType: "xfs"},
			{DevicePath: "/dev/vda2", MountPoint: "/boot/efi", FSType: "vfat"},
		}},
	}
	got := mountEntriesFromDiskInfos("/tmp/kc-guest", infos)
	want := []mountEntry{
		{Device: "/dev/vda3", AppliancePath: "/mnt/guest"},
		{Device: "/dev/vda4", AppliancePath: "/mnt/guest/boot"},
		{Device: "/dev/vda2", AppliancePath: "/mnt/guest/boot/efi"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("mountEntriesFromDiskInfos = %+v, want %+v", got, want)
	}

	escaped := []types.DiskInfo{
		{Partitions: []types.PartitionInfo{
			{DevicePath: "/dev/vda1", MountPoint: "/", FSType: "xfs"},
			{DevicePath: "/dev/vda9", MountPoint: "/../../proc", FSType: "xfs"},
		}},
	}
	got = mountEntriesFromDiskInfos("/tmp/kc-guest", escaped)
	var sawConfined bool
	for _, e := range got {
		if e.AppliancePath == "/proc" || strings.HasPrefix(e.AppliancePath, "/proc/") {
			t.Fatalf("traversal mount recorded as %q", e.AppliancePath)
		}
		if e.Device == "/dev/vda9" && e.AppliancePath == "/mnt/guest/proc" {
			sawConfined = true
		}
	}
	if !sawConfined {
		t.Fatalf("expected confined /mnt/guest/proc entry, got %+v", got)
	}
}
