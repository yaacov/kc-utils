//go:build linux

package grub2

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectWithDefaultGrub(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "default"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "default", "grub"), []byte("GRUB_TIMEOUT=5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := &Grub2Handler{}
	if !g.Detect(root) {
		t.Error("Detect = false, want true with etc/default/grub")
	}
}

func TestDetectWithGrub2Cfg(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "boot", "grub2"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "boot", "grub2", "grub.cfg"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	g := &Grub2Handler{}
	if !g.Detect(root) {
		t.Error("Detect = false, want true with boot/grub2/grub.cfg")
	}
}

func TestDetectWithGrubCfg(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "boot", "grub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "boot", "grub", "grub.cfg"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	g := &Grub2Handler{}
	if !g.Detect(root) {
		t.Error("Detect = false, want true with boot/grub/grub.cfg")
	}
}

func TestDetectAbsent(t *testing.T) {
	root := t.TempDir()
	g := &Grub2Handler{}
	if g.Detect(root) {
		t.Error("Detect = true, want false on empty dir")
	}
}

func TestGetDefaultKernel(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "default"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "default", "grub"), []byte("GRUB_DEFAULT=0\nGRUB_TIMEOUT=5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := &Grub2Handler{}
	got, err := g.GetDefaultKernel(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != "0" {
		t.Errorf("GetDefaultKernel = %q, want 0", got)
	}
}

func TestGetDefaultKernelMissing(t *testing.T) {
	root := t.TempDir()
	g := &Grub2Handler{}
	_, err := g.GetDefaultKernel(root)
	if err == nil {
		t.Error("GetDefaultKernel should error when etc/default/grub missing")
	}
}

func TestAddKernelArg(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "default"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "default", "grub"),
		[]byte("GRUB_CMDLINE_LINUX=\"root=/dev/sda1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := &Grub2Handler{}
	if err := g.AddKernelArg(root, "console=ttyS0"); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(root, "etc", "default", "grub"))
	if !strings.Contains(string(content), "console=ttyS0") {
		t.Error("console=ttyS0 not added")
	}
}

func TestRemoveKernelArg(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "default"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "default", "grub"),
		[]byte("GRUB_CMDLINE_LINUX=\"rhgb quiet root=/dev/sda1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := &Grub2Handler{}
	if err := g.RemoveKernelArg(root, "rhgb"); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(root, "etc", "default", "grub"))
	if strings.Contains(string(content), "rhgb") {
		t.Error("rhgb should be removed")
	}
	if !strings.Contains(string(content), "root=/dev/sda1") {
		t.Error("root=/dev/sda1 should be preserved")
	}
}

func TestAddKernelArgMissingFile(t *testing.T) {
	root := t.TempDir()
	g := &Grub2Handler{}
	if err := g.AddKernelArg(root, "console=ttyS0"); err == nil {
		t.Error("AddKernelArg should error when etc/default/grub missing")
	}
}

func TestRemoveKernelArgMissingFile(t *testing.T) {
	root := t.TempDir()
	g := &Grub2Handler{}
	if err := g.RemoveKernelArg(root, "rhgb"); err == nil {
		t.Error("RemoveKernelArg should error when etc/default/grub missing")
	}
}
