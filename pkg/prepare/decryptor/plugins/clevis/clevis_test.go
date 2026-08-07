//go:build linux

package clevis

import (
	"strings"
	"testing"
)

func TestMapperNameDistinctForSameBasename(t *testing.T) {
	a := mapperName("/dev/mapper/luks-root")
	b := mapperName("/dev/disk/by-id/luks-root")
	if a == b {
		t.Fatalf("expected distinct mapper names for same basename, got %q for both", a)
	}
	if !strings.HasPrefix(a, "v2v-luks-clevis-luks-root-") {
		t.Fatalf("mapper %q missing sanitized basename prefix", a)
	}
	if !strings.HasPrefix(b, "v2v-luks-clevis-luks-root-") {
		t.Fatalf("mapper %q missing sanitized basename prefix", b)
	}
	if mapperName("/dev/mapper/luks-root") != a {
		t.Fatal("mapperName must be deterministic for the same device path")
	}
}
