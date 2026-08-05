//go:build linux

package hyperv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectPresent(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "usr", "sbin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "usr", "sbin", "hv_kvp_daemon"), nil, 0o755); err != nil {
		t.Fatal(err)
	}
	u := &Cleanup{}
	if !u.Detect(root) {
		t.Error("Detect = false, want true with hv_kvp_daemon")
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
	svc := filepath.Join(wantsDir, "hv-kvp-daemon.service")
	if err := os.WriteFile(svc, []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	u := &Cleanup{}
	if err := u.Cleanup(root); err != nil {
		t.Fatalf("Cleanup error: %v", err)
	}
	if _, err := os.Stat(svc); !os.IsNotExist(err) {
		t.Error("hv-kvp-daemon.service should be removed")
	}
}
