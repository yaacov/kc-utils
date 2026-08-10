//go:build linux

package hypervisor

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
