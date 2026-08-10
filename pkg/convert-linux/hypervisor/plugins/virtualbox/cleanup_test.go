//go:build linux

package virtualbox

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor/plugins/testassert"
)

func TestDetectUsrBin(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "usr", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "usr", "bin", "VBoxService"), nil, 0o755); err != nil {
		t.Fatal(err)
	}
	u := &Cleanup{}
	if !u.Detect(root) {
		t.Error("Detect = false, want true with usr/bin/VBoxService")
	}
}

func TestDetectUsrSbin(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "usr", "sbin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "usr", "sbin", "VBoxService"), nil, 0o755); err != nil {
		t.Fatal(err)
	}
	u := &Cleanup{}
	if !u.Detect(root) {
		t.Error("Detect = false, want true with usr/sbin/VBoxService")
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
	svc1 := filepath.Join(wantsDir, "vboxadd-service.service")
	svc2 := filepath.Join(wantsDir, "vboxadd.service")
	if err := os.WriteFile(svc1, []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(svc2, []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	vendorWantsDir := filepath.Join(root, "usr", "lib", "systemd", "system", "multi-user.target.wants")
	if err := os.MkdirAll(vendorWantsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorWantsDir, "vboxadd-service.service"), []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorWantsDir, "vboxadd.service"), []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	u := &Cleanup{}
	if err := u.Cleanup(root); err != nil {
		t.Fatalf("Cleanup error: %v", err)
	}
	if _, err := os.Stat(svc1); !os.IsNotExist(err) {
		t.Error("vboxadd-service.service should be removed")
	}
	if _, err := os.Stat(svc2); !os.IsNotExist(err) {
		t.Error("vboxadd.service should be removed")
	}
	testassert.UnitDisabled(t, root, "vboxadd-service.service")
	testassert.UnitDisabled(t, root, "vboxadd.service")
}
