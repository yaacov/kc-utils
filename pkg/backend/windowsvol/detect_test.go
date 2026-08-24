package windowsvol

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/backend"
	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		partType, fsType string
		want             Kind
		ok               bool
	}{
		{"", "BitLocker", KindBitLocker, true},
		{gptLDMMetadata, "", KindLDM, true},
		{gptLDMData, "", KindLDM, true},
		{"0x42", "", KindLDM, true},
		{gptStorageSpacesProtective, "", KindStorageSpaces, true},
		{"e3c9e316-0b5c-4db8-817d-f92df00215ae", "", "", false},
		{"C12A7328-F81F-11D2-BA4B-00A0C93EC93B", "vfat", "", false},
	}
	for _, c := range cases {
		got, ok := Classify(c.partType, c.fsType)
		if ok != c.ok || got != c.want {
			t.Errorf("Classify(%q, %q) = (%q, %v), want (%q, %v)", c.partType, c.fsType, got, ok, c.want, c.ok)
		}
	}
}

func TestFirstUnsupported(t *testing.T) {
	issues := []Issue{
		{Kind: KindLDM, Device: "/dev/vda2"},
	}
	if issue := FirstUnsupported(issues, backend.NameDirect); issue == nil || issue.Kind != KindLDM {
		t.Fatalf("direct should reject LDM, got %+v", issue)
	}
	if issue := FirstUnsupported(issues, backend.NameQEMU); issue != nil {
		t.Fatalf("qemu should accept LDM metadata, got %+v", issue)
	}
	ss := []Issue{{Kind: KindStorageSpaces, Device: "/dev/vda3"}}
	if issue := FirstUnsupported(ss, backend.NameQEMU); issue == nil {
		t.Fatal("qemu should reject storage spaces")
	}
}

func TestScanDiskInfos(t *testing.T) {
	disks := []types.DiskInfo{{
		Partitions: []types.PartitionInfo{
			{DevicePath: "/dev/vda1", FSType: "vfat"},
			{DevicePath: "/dev/vda2", FSType: "bitlocker"},
		},
	}}
	issues := ScanDiskInfos(disks, nil)
	if len(issues) != 1 || issues[0].Kind != KindBitLocker {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestUnsupportedError(t *testing.T) {
	err := UnsupportedError(KindBitLocker, "/dev/vda2", backend.NameGuestfs)
	if err == nil || err.Error() == "" {
		t.Fatal("expected error message")
	}
}
