package copy

import (
	"strings"
	"testing"
)

func TestNormalizeDiskPath(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"[ds] vm/disk-000001.vmdk", "[ds] vm/disk.vmdk"},
		{"[ds] vm/disk.vmdk", "[ds] vm/disk.vmdk"},
		{"[ds] vm/disk-00001.vmdk", "[ds] vm/disk-00001.vmdk"}, // not six digits
		{"no-suffix", "no-suffix"},
	}
	for _, tc := range cases {
		if got := normalizeDiskPath(tc.in); got != tc.want {
			t.Errorf("normalizeDiskPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFilterDiskURLsOrderAndDeltaMatch(t *testing.T) {
	lease := []DiskURL{
		{URL: "http://lease/b", DiskPath: "[ds] vm/b-000001.vmdk", Size: 2},
		{URL: "http://lease/a", DiskPath: "[ds] vm/a.vmdk", Size: 1},
		{URL: "http://lease/skip", DiskPath: "[ds] vm/shared.vmdk", Size: 3},
	}
	sources := []string{"[ds] vm/a.vmdk", "[ds] vm/b.vmdk"}
	got, err := FilterDiskURLs(lease, sources)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].URL != "http://lease/a" || got[1].URL != "http://lease/b" {
		t.Fatalf("order/match wrong: %+v", got)
	}
}

func TestFilterDiskURLsMissing(t *testing.T) {
	lease := []DiskURL{{URL: "http://lease/a", DiskPath: "[ds] vm/a.vmdk"}}
	_, err := FilterDiskURLs(lease, []string{"[ds] vm/missing.vmdk"})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestFilterDiskURLsEmptySources(t *testing.T) {
	lease := []DiskURL{
		{URL: "http://lease/a", DiskPath: "[ds] vm/a.vmdk"},
		{URL: "http://lease/b", DiskPath: "[ds] vm/b.vmdk"},
	}
	got, err := FilterDiskURLs(lease, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].URL != "http://lease/a" || got[1].URL != "http://lease/b" {
		t.Fatalf("empty sources should copy all lease disks, got %+v", got)
	}

	got, err = FilterDiskURLs(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("empty lease: got %d, want 0", len(got))
	}
}
