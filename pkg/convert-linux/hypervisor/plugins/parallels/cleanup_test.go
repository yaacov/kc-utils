//go:build linux

package parallels

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectPrlsrvctl(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "usr", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "usr", "bin", "prlsrvctl"), nil, 0o755); err != nil {
		t.Fatal(err)
	}
	u := &Cleanup{}
	if !u.Detect(root) {
		t.Error("Detect = false, want true with prlsrvctl")
	}
}

func TestDetectPrltoolsd(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "usr", "sbin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "usr", "sbin", "prltoolsd"), nil, 0o755); err != nil {
		t.Fatal(err)
	}
	u := &Cleanup{}
	if !u.Detect(root) {
		t.Error("Detect = false, want true with prltoolsd")
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
	svc := filepath.Join(wantsDir, "prltoolsd.service")
	if err := os.WriteFile(svc, []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	u := &Cleanup{}
	if err := u.Cleanup(root); err != nil {
		t.Fatalf("Cleanup error: %v", err)
	}
	if _, err := os.Stat(svc); !os.IsNotExist(err) {
		t.Error("prltoolsd.service should be removed")
	}
}
