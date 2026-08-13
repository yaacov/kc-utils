//go:build linux

package systemd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDisableSystemdUnit(t *testing.T) {
	root := t.TempDir()
	unit := "example.service"

	wantsDirs := []string{
		filepath.Join(root, "etc", "systemd", "system", "multi-user.target.wants"),
		filepath.Join(root, "etc", "systemd", "system", "default.target.wants"),
		filepath.Join(root, "etc", "systemd", "system", "sockets.target.wants"),
		filepath.Join(root, "etc", "systemd", "system", "graphical.target.wants"),
		filepath.Join(root, "usr", "lib", "systemd", "system", "multi-user.target.wants"),
		filepath.Join(root, "usr", "lib", "systemd", "system", "default.target.wants"),
		filepath.Join(root, "usr", "lib", "systemd", "system", "sockets.target.wants"),
		filepath.Join(root, "usr", "lib", "systemd", "system", "graphical.target.wants"),
	}
	for _, dir := range wantsDirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(dir, unit)
		if err := os.WriteFile(link, []byte("enabled"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	DisableSystemdUnit(root, unit)

	for _, dir := range wantsDirs {
		link := filepath.Join(dir, unit)
		if _, err := os.Stat(link); !os.IsNotExist(err) {
			t.Errorf("wants symlink %s still exists", link)
		}
	}

	mask := filepath.Join(root, "etc", "systemd", "system", unit)
	target, err := os.Readlink(mask)
	if err != nil {
		t.Fatalf("reading unit mask symlink: %v", err)
	}
	if target != "/dev/null" {
		t.Errorf("mask target = %q, want /dev/null", target)
	}
}

func TestUnitIsMasked(t *testing.T) {
	root := t.TempDir()
	unitDir := filepath.Join(root, "etc", "systemd", "system")
	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	unit := "NetworkManager.service"
	maskPath := filepath.Join(unitDir, unit)
	if err := os.Symlink(UnitMaskTarget, maskPath); err != nil {
		t.Fatal(err)
	}
	if !UnitIsMasked(root, unit) {
		t.Error("UnitIsMasked = false, want true")
	}
}

func TestDisableEC2NetHooks(t *testing.T) {
	root := t.TempDir()
	wantsDir := filepath.Join(root, "etc", "systemd", "system", "multi-user.target.wants")
	if err := os.MkdirAll(wantsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, unit := range []string{
		"set-hostname-imds.service",
		"policy-routes@ens5.service",
		"refresh-policy-routes@ens5.timer",
	} {
		if err := os.WriteFile(filepath.Join(wantsDir, unit), []byte("fake"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	DisableEC2NetHooks(root)

	for _, unit := range []string{
		"set-hostname-imds.service",
		"policy-routes@ens5.service",
		"refresh-policy-routes@ens5.timer",
	} {
		mask := filepath.Join(root, "etc", "systemd", "system", unit)
		target, err := os.Readlink(mask)
		if err != nil {
			t.Fatalf("reading mask for %s: %v", unit, err)
		}
		if target != UnitMaskTarget {
			t.Errorf("mask target for %s = %q, want %q", unit, target, UnitMaskTarget)
		}
	}
}

func TestUnitWantsEnabledAbsoluteSymlink(t *testing.T) {
	root := t.TempDir()
	wantsDir := filepath.Join(root, "etc", "systemd", "system", "multi-user.target.wants")
	if err := os.MkdirAll(wantsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	unit := "example.service"
	// Absolute symlink target resolved against the guest root, not the host
	// filesystem: os.Stat on the link itself would fail to find it under /,
	// so UnitWantsEnabled must detect the symlink entry without dereferencing it.
	if err := os.Symlink("/usr/lib/systemd/system/example.service", filepath.Join(wantsDir, unit)); err != nil {
		t.Fatal(err)
	}
	if !UnitWantsEnabled(root, unit) {
		t.Error("UnitWantsEnabled = false, want true for real absolute-target symlink")
	}
}

func TestUnitWantsEnabledGraphicalWants(t *testing.T) {
	root := t.TempDir()
	wantsDir := filepath.Join(root, "usr", "lib", "systemd", "system", "graphical.target.wants")
	if err := os.MkdirAll(wantsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	unit := "example.service"
	if err := os.WriteFile(filepath.Join(wantsDir, unit), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !UnitWantsEnabled(root, unit) {
		t.Error("UnitWantsEnabled = false, want true for graphical.target.wants entry")
	}
}

func TestUnitWantsEnabledFalse(t *testing.T) {
	root := t.TempDir()
	if UnitWantsEnabled(root, "missing.service") {
		t.Error("UnitWantsEnabled = true, want false when no wants entry exists")
	}
}

func TestDisableEC2NetHooksVendorAndNonMultiUserWants(t *testing.T) {
	root := t.TempDir()
	vendorWantsDir := filepath.Join(root, "usr", "lib", "systemd", "system", "multi-user.target.wants")
	defaultWantsDir := filepath.Join(root, "etc", "systemd", "system", "default.target.wants")
	for _, dir := range []string{vendorWantsDir, defaultWantsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(vendorWantsDir, "policy-routes@ens5.service"), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(defaultWantsDir, "refresh-policy-routes@ens5.timer"), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	DisableEC2NetHooks(root)

	for _, unit := range []string{
		"policy-routes@ens5.service",
		"refresh-policy-routes@ens5.timer",
	} {
		mask := filepath.Join(root, "etc", "systemd", "system", unit)
		target, err := os.Readlink(mask)
		if err != nil {
			t.Fatalf("reading mask for %s: %v", unit, err)
		}
		if target != UnitMaskTarget {
			t.Errorf("mask target for %s = %q, want %q", unit, target, UnitMaskTarget)
		}
	}
}

func writeUnitFile(t *testing.T, root, unit string) {
	t.Helper()
	unitDir := filepath.Join(root, "usr", "lib", "systemd", "system")
	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unitDir, unit), []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeAdminUnitFile(t *testing.T, root, unit string) {
	t.Helper()
	unitDir := filepath.Join(root, "etc", "systemd", "system")
	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unitDir, unit), []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestEnableSystemdUnitDisabled(t *testing.T) {
	root := t.TempDir()
	unit := "example.service"
	writeUnitFile(t, root, unit)

	if err := EnableSystemdUnit(root, unit); err != nil {
		t.Fatalf("EnableSystemdUnit returned error: %v", err)
	}
	if !UnitWantsEnabled(root, unit) {
		t.Error("UnitWantsEnabled = false, want true after EnableSystemdUnit")
	}
	wantsLink := filepath.Join(root, "etc", "systemd", "system", "multi-user.target.wants", unit)
	target, err := os.Readlink(wantsLink)
	if err != nil {
		t.Fatalf("reading wants symlink: %v", err)
	}
	if target != "/usr/lib/systemd/system/example.service" {
		t.Errorf("wants target = %q, want /usr/lib/systemd/system/example.service", target)
	}
}

func TestEnableSystemdUnitMasked(t *testing.T) {
	root := t.TempDir()
	unit := "example.service"
	writeUnitFile(t, root, unit)

	maskDir := filepath.Join(root, "etc", "systemd", "system")
	if err := os.MkdirAll(maskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(UnitMaskTarget, filepath.Join(maskDir, unit)); err != nil {
		t.Fatal(err)
	}

	if err := EnableSystemdUnit(root, unit); err != nil {
		t.Fatalf("EnableSystemdUnit returned error: %v", err)
	}
	if UnitIsMasked(root, unit) {
		t.Error("UnitIsMasked = true, want false after EnableSystemdUnit")
	}
	if !UnitWantsEnabled(root, unit) {
		t.Error("UnitWantsEnabled = false, want true after EnableSystemdUnit")
	}
}

func TestEnableSystemdUnitAlreadyEnabled(t *testing.T) {
	root := t.TempDir()
	unit := "example.service"
	writeUnitFile(t, root, unit)

	vendorWantsDir := filepath.Join(root, "usr", "lib", "systemd", "system", "multi-user.target.wants")
	if err := os.MkdirAll(vendorWantsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/usr/lib/systemd/system/example.service", filepath.Join(vendorWantsDir, unit)); err != nil {
		t.Fatal(err)
	}

	if err := EnableSystemdUnit(root, unit); err != nil {
		t.Fatalf("EnableSystemdUnit returned error: %v", err)
	}
	adminWants := filepath.Join(root, "etc", "systemd", "system", "multi-user.target.wants", unit)
	if _, err := os.Lstat(adminWants); !os.IsNotExist(err) {
		t.Error("admin wants symlink created when vendor wants already enabled")
	}
}

func TestEnableSystemdUnitMissingUnitFile(t *testing.T) {
	root := t.TempDir()
	if err := EnableSystemdUnit(root, "missing.service"); err == nil {
		t.Fatal("EnableSystemdUnit = nil, want error when unit file is missing")
	}
}

func TestEnableSystemdUnitEtcOnlyUnitFile(t *testing.T) {
	root := t.TempDir()
	unit := "example.service"
	writeAdminUnitFile(t, root, unit)

	if err := EnableSystemdUnit(root, unit); err != nil {
		t.Fatalf("EnableSystemdUnit returned error: %v", err)
	}
	if !UnitWantsEnabled(root, unit) {
		t.Error("UnitWantsEnabled = false, want true after EnableSystemdUnit")
	}
	wantsLink := filepath.Join(root, "etc", "systemd", "system", "multi-user.target.wants", unit)
	target, err := os.Readlink(wantsLink)
	if err != nil {
		t.Fatalf("reading wants symlink: %v", err)
	}
	if target != "/etc/systemd/system/example.service" {
		t.Errorf("wants target = %q, want /etc/systemd/system/example.service", target)
	}
}
