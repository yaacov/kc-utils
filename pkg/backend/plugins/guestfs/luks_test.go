//go:build unix

package guestfs

import (
	"testing"
)

func TestQuoteGuestfishClevisArgs(t *testing.T) {
	got := "clevis-luks-unlock " + quoteGuestfish("/dev/sda2") + " " + quoteGuestfish("v2v-luks-clevis-sda2")
	want := "clevis-luks-unlock /dev/sda2 v2v-luks-clevis-sda2"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	spaced := quoteGuestfish("/dev/mapper/my vol")
	if spaced != `"/dev/mapper/my vol"` {
		t.Fatalf("spaced quote = %q", spaced)
	}
}

func TestDmOutputContains(t *testing.T) {
	tests := []struct {
		name   string
		output string
		mapped string
		want   bool
	}{
		{"exact match", "/dev/mapper/v2v-luks-clevis-sda2\n", "/dev/mapper/v2v-luks-clevis-sda2", true},
		{"basename match", "v2v-luks-clevis-sda2\n", "/dev/mapper/v2v-luks-clevis-sda2", true},
		{"no match", "/dev/mapper/other\n", "/dev/mapper/v2v-luks-clevis-sda2", false},
		{"empty output", "", "/dev/mapper/x", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dmOutputContains(tc.output, tc.mapped)
			if got != tc.want {
				t.Fatalf("dmOutputContains(%q, %q) = %v, want %v", tc.output, tc.mapped, got, tc.want)
			}
		})
	}
}
