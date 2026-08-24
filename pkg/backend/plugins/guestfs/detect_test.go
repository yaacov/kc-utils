//go:build unix

package guestfs

import (
	"reflect"
	"testing"
)

func TestParseListPartitions(t *testing.T) {
	out := "/dev/sda1\n/dev/sda2\n/dev/nvme0n1p1\n/dev/nvme0n1p2\n\n"
	got := parseListPartitions(out)
	want := []string{"/dev/sda1", "/dev/sda2", "/dev/nvme0n1p1", "/dev/nvme0n1p2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseListPartitions = %v, want %v", got, want)
	}
}

func TestPartitionsForDisk(t *testing.T) {
	all := parseListPartitions("/dev/sda1\n/dev/sda2\n/dev/nvme0n1p2\n")
	cases := []struct {
		disk string
		want []string
	}{
		{"/dev/sda", []string{"/dev/sda1", "/dev/sda2"}},
		{"/dev/nvme0n1", []string{"/dev/nvme0n1p2"}},
	}
	for _, c := range cases {
		got := partitionsForDisk(c.disk, all)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("partitionsForDisk(%q) = %v, want %v", c.disk, got, c.want)
		}
	}
}
