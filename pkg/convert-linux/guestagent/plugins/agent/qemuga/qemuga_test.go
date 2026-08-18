//go:build unix

package qemuga

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectPresent(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "usr", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "qemu-ga"), []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	a := &QEMUAgent{}
	if !a.Detect(root) {
		t.Errorf("Detect = false, want true when qemu-ga binary exists")
	}
}

func TestDetectAbsent(t *testing.T) {
	root := t.TempDir()

	a := &QEMUAgent{}
	if a.Detect(root) {
		t.Errorf("Detect = true, want false on empty dir")
	}
}

func TestRemove(t *testing.T) {
	root := t.TempDir()

	binDir := filepath.Join(root, "usr", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "qemu-ga"), []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	serviceDir := filepath.Join(root, "usr", "lib", "systemd", "system")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "qemu-guest-agent.service"), []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &QEMUAgent{}
	if err := a.Remove(root); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(binDir, "qemu-ga")); !os.IsNotExist(err) {
		t.Errorf("qemu-ga binary still exists after Remove")
	}
	if _, err := os.Stat(filepath.Join(serviceDir, "qemu-guest-agent.service")); !os.IsNotExist(err) {
		t.Errorf("service file still exists after Remove")
	}
}

func TestRemoveMissingFiles(t *testing.T) {
	root := t.TempDir()

	a := &QEMUAgent{}
	if err := a.Remove(root); err != nil {
		t.Fatalf("Remove returned error on empty dir: %v", err)
	}
}
