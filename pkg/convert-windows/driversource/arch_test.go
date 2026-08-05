package driversource

import "testing"

func TestNormalizeArch(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"x86_64", "amd64"},
		{"amd64", "amd64"},
		{"X64", "amd64"},
		{"i386", "x86"},
		{"x86", "x86"},
		{"aarch64", "arm64"},
	}
	for _, tc := range tests {
		if got := NormalizeArch(tc.in); got != tc.want {
			t.Errorf("NormalizeArch(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestArchMatches(t *testing.T) {
	if !ArchMatches("amd64", "x86_64") {
		t.Error("amd64 should match x86_64")
	}
	if ArchMatches("x86", "x86_64") {
		t.Error("x86 should not match x86_64")
	}
}
