//go:build linux

package kudzu

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor/plugins/testassert"
)

func TestDetectPresent(t *testing.T) {
	root := t.TempDir()
	initScript := filepath.Join(root, "etc", "init.d", "kudzu")
	if err := os.MkdirAll(filepath.Dir(initScript), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(initScript, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	u := &Cleanup{}
	if !u.Detect(root) {
		t.Error("Detect = false, want true when kudzu init script exists")
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
	unit := "kudzu.service"

	rcLink := filepath.Join(root, "etc", "rc3.d", "S06kudzu")
	if err := os.MkdirAll(filepath.Dir(rcLink), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rcLink, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	wantsDir := filepath.Join(root, "etc", "systemd", "system", "multi-user.target.wants")
	if err := os.MkdirAll(wantsDir, 0o755); err != nil {
		t.Fatal(err)
	}
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

	if _, err := os.Stat(rcLink); !os.IsNotExist(err) {
		t.Error("kudzu rc.d symlink still exists after Cleanup")
	}
	if _, err := os.Stat(svc); !os.IsNotExist(err) {
		t.Error("kudzu.service wants symlink still exists after Cleanup")
	}
	testassert.UnitDisabled(t, root, unit)
}
