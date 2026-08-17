//go:build linux

package guest

import (
	"testing"
)

func TestSanitizeDevice(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"/dev/sda1", "_dev_sda1"},
		{"nvme0n1p3", "nvme0n1p3"},
		{"", ""},
		{"/dev/mapper/luks-abc", "_dev_mapper_luks_abc"},
	}
	for _, tc := range cases {
		if got := sanitizeDevice(tc.input); got != tc.want {
			t.Errorf("sanitizeDevice(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestStringsTrimGuest(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"/etc/fstab", "etc/fstab"},
		{"etc/fstab", "etc/fstab"},
		{"/", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := stringsTrimGuest(tc.input); got != tc.want {
			t.Errorf("stringsTrimGuest(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
