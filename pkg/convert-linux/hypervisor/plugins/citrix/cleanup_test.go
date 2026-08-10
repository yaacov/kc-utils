//go:build linux

package citrix

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor/plugins/testassert"
)

func TestDetectXenInventory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "xensource-inventory"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	u := &Cleanup{}
	if !u.Detect(root) {
		t.Error("Detect = false, want true with xensource-inventory")
	}
}

func TestDetectXeDaemon(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "usr", "sbin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "usr", "sbin", "xe-daemon"), nil, 0o755); err != nil {
		t.Fatal(err)
	}
	u := &Cleanup{}
	if !u.Detect(root) {
		t.Error("Detect = false, want true with xe-daemon")
	}
}

func TestDetectAbsent(t *testing.T) {
	root := t.TempDir()
	u := &Cleanup{}
	if u.Detect(root) {
		t.Error("Detect = true, want false on empty dir")
	}
}

func TestCleanup(t *testing.T) {
	root := t.TempDir()
	wantsDir := filepath.Join(root, "etc", "systemd", "system", "multi-user.target.wants")
	if err := os.MkdirAll(wantsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	unit := "xe-daemon.service"
	svc := filepath.Join(wantsDir, unit)
	if err := os.WriteFile(svc, []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	vendorWantsDir := filepath.Join(root, "usr", "lib", "systemd", "system", "multi-user.target.wants")
	if err := os.MkdirAll(vendorWantsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorWantsDir, unit), []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	u := &Cleanup{}
	if err := u.Cleanup(root); err != nil {
		t.Fatalf("Cleanup error: %v", err)
	}
	if _, err := os.Stat(svc); !os.IsNotExist(err) {
		t.Error("xe-daemon.service should be removed")
	}
	testassert.UnitDisabled(t, root, unit)
	testassert.UnitDisabled(t, root, "xe-linux-distribution.service")
}

func TestCleanupAlreadyAbsent(t *testing.T) {
	root := t.TempDir()
	u := &Cleanup{}
	if err := u.Cleanup(root); err != nil {
		t.Fatalf("Cleanup should not error when service missing: %v", err)
	}
}
