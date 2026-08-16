//go:build linux

package guest

import "testing"

func TestNormalizeGuestPath(t *testing.T) {
	if normalizeGuestPath("etc/fstab") != "/etc/fstab" {
		t.Fatalf("got %q", normalizeGuestPath("etc/fstab"))
	}
	if normalizeGuestPath("/etc/../etc/fstab") != "/etc/fstab" {
		t.Fatalf("got %q", normalizeGuestPath("/etc/../etc/fstab"))
	}
}
