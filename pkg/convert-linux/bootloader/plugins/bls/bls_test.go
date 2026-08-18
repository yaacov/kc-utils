//go:build unix

package bls

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeEntries(t *testing.T, root string, files map[string]string) {
	t.Helper()
	dir := filepath.Join(root, "boot", "loader", "entries")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDetectWithConf(t *testing.T) {
	root := t.TempDir()
	makeEntries(t, root, map[string]string{
		"5.14.0-284.conf": "title RHEL\nlinux /vmlinuz-5.14.0-284\noptions root=/dev/sda1\n",
	})
	b := &BLSHandler{}
	if !b.Detect(root) {
		t.Error("Detect = false, want true")
	}
}

func TestDetectEmpty(t *testing.T) {
	root := t.TempDir()
	b := &BLSHandler{}
	if b.Detect(root) {
		t.Error("Detect = true, want false on empty dir")
	}
}

func TestDetectNoConf(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "boot", "loader", "entries")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("not a conf"), 0o644); err != nil {
		t.Fatal(err)
	}
	b := &BLSHandler{}
	if b.Detect(root) {
		t.Error("Detect = true with no .conf files")
	}
}

func TestGetDefaultKernel(t *testing.T) {
	root := t.TempDir()
	makeEntries(t, root, map[string]string{
		"5.14.0-284.conf": "title RHEL\nlinux /vmlinuz-5.14.0-284\noptions root=/dev/sda1\n",
	})
	b := &BLSHandler{}
	got, err := b.GetDefaultKernel(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != "/vmlinuz-5.14.0-284" {
		t.Errorf("GetDefaultKernel = %q, want /vmlinuz-5.14.0-284", got)
	}
}

func TestGetDefaultKernelMissing(t *testing.T) {
	root := t.TempDir()
	b := &BLSHandler{}
	_, err := b.GetDefaultKernel(root)
	if err == nil {
		t.Error("GetDefaultKernel should error when no entries dir")
	}
}

func TestAddKernelArg(t *testing.T) {
	root := t.TempDir()
	makeEntries(t, root, map[string]string{
		"5.14.0-284.conf": "title RHEL\nlinux /vmlinuz-5.14.0-284\noptions root=/dev/sda1\n",
	})
	b := &BLSHandler{}
	if err := b.AddKernelArg(root, "console=ttyS0"); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(root, "boot", "loader", "entries", "5.14.0-284.conf"))
	if !strings.Contains(string(content), "console=ttyS0") {
		t.Error("console=ttyS0 not added to options")
	}
}

func TestAddKernelArgNoDuplicate(t *testing.T) {
	root := t.TempDir()
	makeEntries(t, root, map[string]string{
		"5.14.0-284.conf": "title RHEL\nlinux /vmlinuz-5.14.0-284\noptions root=/dev/sda1 console=ttyS0\n",
	})
	b := &BLSHandler{}
	if err := b.AddKernelArg(root, "console=ttyS0"); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(root, "boot", "loader", "entries", "5.14.0-284.conf"))
	count := strings.Count(string(content), "console=ttyS0")
	if count != 1 {
		t.Errorf("console=ttyS0 appears %d times, want 1", count)
	}
}

func TestRemoveKernelArgExact(t *testing.T) {
	root := t.TempDir()
	makeEntries(t, root, map[string]string{
		"5.14.0-284.conf": "title RHEL\nlinux /vmlinuz-5.14.0-284\noptions rhgb quiet root=/dev/sda1\n",
	})
	b := &BLSHandler{}
	if err := b.RemoveKernelArg(root, "rhgb"); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(root, "boot", "loader", "entries", "5.14.0-284.conf"))
	if strings.Contains(string(content), "rhgb") {
		t.Error("rhgb should be removed")
	}
	if !strings.Contains(string(content), "root=/dev/sda1") {
		t.Error("root=/dev/sda1 should be preserved")
	}
}

func TestRemoveKernelArgPrefix(t *testing.T) {
	root := t.TempDir()
	makeEntries(t, root, map[string]string{
		"5.14.0-284.conf": "title RHEL\nlinux /vmlinuz-5.14.0-284\noptions vga=normal root=/dev/sda1\n",
	})
	b := &BLSHandler{}
	if err := b.RemoveKernelArg(root, "vga"); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(root, "boot", "loader", "entries", "5.14.0-284.conf"))
	if strings.Contains(string(content), "vga") {
		t.Error("vga=normal should be removed")
	}
}

func TestModifyMultipleEntries(t *testing.T) {
	root := t.TempDir()
	makeEntries(t, root, map[string]string{
		"5.14.0-284.conf": "title RHEL\nlinux /vmlinuz-5.14.0-284\noptions root=/dev/sda1\n",
		"4.18.0-305.conf": "title RHEL-old\nlinux /vmlinuz-4.18.0-305\noptions root=/dev/sda1\n",
	})
	b := &BLSHandler{}
	if err := b.AddKernelArg(root, "console=ttyS0"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"5.14.0-284.conf", "4.18.0-305.conf"} {
		content, _ := os.ReadFile(filepath.Join(root, "boot", "loader", "entries", name))
		if !strings.Contains(string(content), "console=ttyS0") {
			t.Errorf("%s: console=ttyS0 not added", name)
		}
	}
}
