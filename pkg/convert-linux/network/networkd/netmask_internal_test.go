//go:build linux

package networkd

import "testing"

func TestNetmaskPrefixValid(t *testing.T) {
	prefix, err := netmaskPrefix("255.255.255.0")
	if err != nil {
		t.Fatalf("netmaskPrefix: %v", err)
	}
	if prefix != "24" {
		t.Errorf("prefix = %q, want 24", prefix)
	}
}

func TestNetmaskPrefixNonContiguous(t *testing.T) {
	if _, err := netmaskPrefix("255.0.255.0"); err == nil {
		t.Error("netmaskPrefix = nil error, want error for non-contiguous mask")
	}
}

func TestNetmaskPrefixOutOfRangeOctet(t *testing.T) {
	if _, err := netmaskPrefix("255.255.256.0"); err == nil {
		t.Error("netmaskPrefix = nil error, want error for out-of-range octet")
	}
}
