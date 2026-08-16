//go:build unix

package testassert

import (
	"os"
	"testing"

	"github.com/yaacov/kc-utils/pkg/convert-linux/systemd"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

// UnitDisabled verifies wants symlinks are removed and the unit is masked.
func UnitDisabled(t *testing.T, guestRoot, unit string) {
	t.Helper()

	if _, err := os.Lstat(systemd.VendorWantsPath(guestRoot, unit)); !os.IsNotExist(err) {
		t.Errorf("vendor wants symlink for %s still exists", unit)
	}

	target, err := guestio.FileReadlink(systemd.SystemdUnitMaskPath(guestRoot, unit))
	if err != nil {
		t.Fatalf("reading mask symlink for %s: %v", unit, err)
	}
	if target != systemd.UnitMaskTarget {
		t.Errorf("mask target for %s = %q, want %q", unit, target, systemd.UnitMaskTarget)
	}
}
