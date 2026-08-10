//go:build linux

package xen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectPresent(t *testing.T) {
	root := t.TempDir()
	writeKernelConfig(t, root, `INITRD_MODULES="virtio xennet xenblk"`)

	u := &Cleanup{}
	if !u.Detect(root) {
		t.Error("Detect = false, want true when xen modules are listed")
	}
}

func TestDetectAbsent(t *testing.T) {
	root := t.TempDir()
	writeKernelConfig(t, root, `INITRD_MODULES="virtio_blk virtio_net"`)

	u := &Cleanup{}
	if u.Detect(root) {
		t.Error("Detect = true, want false when no xen modules are listed")
	}
}

func TestCleanupRemovesXenModules(t *testing.T) {
	root := t.TempDir()
	writeKernelConfig(t, root, `INITRD_MODULES="virtio xennet xenblk"
DOMU_INITRD_MODULES="xen-vnif virtio"`)

	u := &Cleanup{}
	if err := u.Cleanup(root); err != nil {
		t.Fatalf("Cleanup error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "etc", "sysconfig", "kernel"))
	if err != nil {
		t.Fatalf("reading kernel config: %v", err)
	}
	content := string(data)
	for _, mod := range []string{"xennet", "xenblk", "xen-vnif"} {
		if strings.Contains(content, mod) {
			t.Errorf("xen module %q still present after Cleanup", mod)
		}
	}
	if !strings.Contains(content, "virtio") {
		t.Error("non-xen module virtio was removed")
	}
}

func writeKernelConfig(t *testing.T, root, content string) {
	t.Helper()
	dir := filepath.Join(root, "etc", "sysconfig")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "kernel"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
