//go:build linux

package nutanix

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectPresent(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "usr", "local", "nutanix", "ngt"), 0o755); err != nil {
		t.Fatal(err)
	}

	u := &Cleanup{}
	if !u.Detect(root) {
		t.Error("Detect = false, want true when NGT dir exists")
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
	svc := filepath.Join(wantsDir, "ngt_guest_agent.service")
	if err := os.WriteFile(svc, []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	initScript := filepath.Join(root, "etc", "rc.d", "init.d", "ngt_guest_agent")
	if err := os.MkdirAll(filepath.Dir(initScript), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(initScript, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	u := &Cleanup{}
	if err := u.Cleanup(root); err != nil {
		t.Fatalf("Cleanup returned error: %v", err)
	}

	if _, err := os.Stat(svc); !os.IsNotExist(err) {
		t.Error("ngt_guest_agent.service still exists after Cleanup")
	}
	if _, err := os.Stat(initScript); !os.IsNotExist(err) {
		t.Error("ngt_guest_agent init script still exists after Cleanup")
	}
}
