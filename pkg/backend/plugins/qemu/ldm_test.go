//go:build unix

package qemu

import (
	"reflect"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestParseLsblkLDMDevices(t *testing.T) {
	out := []byte(`{"blockdevices":[
		{"name":"ldm_Volume1-part1","path":"/dev/mapper/ldm_Volume1-part1","type":"lvm","fstype":"ntfs","size":1048576},
		{"name":"dm-0","path":"/dev/mapper/dm-0","type":"lvm","fstype":"ext4","size":2048},
		{"name":"ldm_Volume2-part1","path":"/dev/mapper/ldm_Volume2-part1","type":"lvm","fstype":"","size":4096}
	]}`)
	parts, paths := parseLsblkLDMDevices(out)
	wantParts := []types.PartitionInfo{
		{Index: 1, DevicePath: "/dev/mapper/ldm_Volume1-part1", FSType: "ntfs", SizeBytes: 1048576},
		{Index: 2, DevicePath: "/dev/mapper/ldm_Volume2-part1", FSType: "", SizeBytes: 4096},
	}
	wantPaths := []string{"/dev/mapper/ldm_Volume1-part1", "/dev/mapper/ldm_Volume2-part1"}
	if !reflect.DeepEqual(parts, wantParts) {
		t.Fatalf("parts = %+v, want %+v", parts, wantParts)
	}
	if !reflect.DeepEqual(paths, wantPaths) {
		t.Fatalf("paths = %v, want %v", paths, wantPaths)
	}
}

func TestParseLsblkLDMDevicesFlatList(t *testing.T) {
	// lsblk -J -l returns a flat device list; LDM mappers are not nested under dm-*.
	out := []byte(`{"blockdevices":[
		{"name":"dm-0","path":"/dev/mapper/dm-0","type":"lvm","fstype":"","size":512},
		{"name":"ldm_Volume1-part1","path":"/dev/mapper/ldm_Volume1-part1","type":"lvm","fstype":"ntfs","size":1048576}
	]}`)
	_, paths := parseLsblkLDMDevices(out)
	if len(paths) != 1 || paths[0] != "/dev/mapper/ldm_Volume1-part1" {
		t.Fatalf("paths = %v", paths)
	}
}

func TestIsLDMMapperName(t *testing.T) {
	if !isLDMMapperName("ldm_Volume1-part1") {
		t.Fatal("expected ldm_ prefix to match")
	}
	if isLDMMapperName("dm-0") {
		t.Fatal("expected non-ldm mapper to be skipped")
	}
}

func TestIsLDMMapperPath(t *testing.T) {
	if !isLDMMapperPath("/dev/mapper/ldm_Volume1-part1") {
		t.Fatal("expected ldm mapper path")
	}
	if isLDMMapperPath("/dev/mapper/cryptroot") {
		t.Fatal("expected non-ldm mapper path to be false")
	}
}
