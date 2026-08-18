//go:build unix

package deb

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanKernelsSorted(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "boot"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, ver := range []string{"5.15.0-91", "6.1.0-18", "5.10.0-28"} {
		if err := os.MkdirAll(filepath.Join(root, "lib", "modules", ver), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "boot", "vmlinuz-"+ver), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "boot", "initrd.img-"+ver), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	s := &Scanner{}
	kernels, err := s.ScanKernels(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(kernels) != 3 {
		t.Fatalf("got %d kernels, want 3", len(kernels))
	}
	if kernels[0].Version != "6.1.0-18" {
		t.Errorf("first kernel = %q, want 6.1.0-18", kernels[0].Version)
	}
	if kernels[2].Version != "5.10.0-28" {
		t.Errorf("last kernel = %q, want 5.10.0-28", kernels[2].Version)
	}
}

func TestScanKernelsDebianNaming(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "lib", "modules", "6.1.0-18"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "boot"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "boot", "vmlinuz-6.1.0-18"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "boot", "initrd.img-6.1.0-18"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	s := &Scanner{}
	kernels, err := s.ScanKernels(root)
	if err != nil {
		t.Fatal(err)
	}
	if kernels[0].Path != "/boot/vmlinuz-6.1.0-18" {
		t.Errorf("Path = %q, want /boot/vmlinuz-6.1.0-18", kernels[0].Path)
	}
	if kernels[0].InitrdPath != "/boot/initrd.img-6.1.0-18" {
		t.Errorf("InitrdPath = %q, want /boot/initrd.img-6.1.0-18", kernels[0].InitrdPath)
	}
}

func TestScanKernelsInitramfsFallback(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "lib", "modules", "6.1.0-18"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "boot"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "boot", "vmlinuz-6.1.0-18"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "boot", "initramfs-6.1.0-18.img"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	s := &Scanner{}
	kernels, err := s.ScanKernels(root)
	if err != nil {
		t.Fatal(err)
	}
	if kernels[0].InitrdPath != "/boot/initramfs-6.1.0-18.img" {
		t.Errorf("InitrdPath = %q, want /boot/initramfs-6.1.0-18.img", kernels[0].InitrdPath)
	}
}

func TestSelectBestRequiresPathOnly(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "lib", "modules", "6.1.0-18"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "boot"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "boot", "vmlinuz-6.1.0-18"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	s := &Scanner{}
	kernels, err := s.ScanKernels(root)
	if err != nil {
		t.Fatal(err)
	}
	best := s.SelectBest(kernels)
	if best == nil {
		t.Fatal("SelectBest returned nil")
	}
	if best.Version != "6.1.0-18" {
		t.Errorf("SelectBest = %q, want 6.1.0-18", best.Version)
	}
	if best.InitrdPath != "" {
		t.Errorf("InitrdPath = %q, expected empty", best.InitrdPath)
	}
}

func TestSelectBestEmpty(t *testing.T) {
	s := &Scanner{}
	if s.SelectBest(nil) != nil {
		t.Error("SelectBest(nil) should return nil")
	}
}

func TestSelectBestFallbackNoPath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "lib", "modules", "6.1.0-18"), 0o755); err != nil {
		t.Fatal(err)
	}

	s := &Scanner{}
	kernels, err := s.ScanKernels(root)
	if err != nil {
		t.Fatal(err)
	}
	best := s.SelectBest(kernels)
	if best == nil {
		t.Fatal("SelectBest returned nil")
	}
	if best.Path != "" {
		t.Errorf("Path = %q, expected empty for fallback", best.Path)
	}
}

func TestScanKernelsMissingDir(t *testing.T) {
	root := t.TempDir()
	s := &Scanner{}
	_, err := s.ScanKernels(root)
	if err == nil {
		t.Error("ScanKernels should error when lib/modules missing")
	}
}

func TestScanKernelsHasVirtio(t *testing.T) {
	root := t.TempDir()
	ver := "6.1.0-18"
	if err := os.MkdirAll(filepath.Join(root, "lib", "modules", ver), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "boot"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "boot", "vmlinuz-"+ver), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	virtioDir := filepath.Join(root, "lib", "modules", ver, "kernel", "drivers", "virtio")
	if err := os.MkdirAll(virtioDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(virtioDir, "virtio_pci.ko.zst"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	s := &Scanner{}
	kernels, err := s.ScanKernels(root)
	if err != nil {
		t.Fatal(err)
	}
	if !kernels[0].HasVirtio {
		t.Error("expected HasVirtio=true when virtio_pci.ko.zst is present")
	}
	if kernels[0].IsXenPV {
		t.Error("expected IsXenPV=false")
	}
}

func TestScanKernelsIsXenPV(t *testing.T) {
	root := t.TempDir()
	ver := "4.19.0-22"
	if err := os.MkdirAll(filepath.Join(root, "lib", "modules", ver), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "boot"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "boot", "vmlinuz-"+ver), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	blockDir := filepath.Join(root, "lib", "modules", ver, "kernel", "drivers", "block")
	if err := os.MkdirAll(blockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(blockDir, "xen-blkfront.ko"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	s := &Scanner{}
	kernels, err := s.ScanKernels(root)
	if err != nil {
		t.Fatal(err)
	}
	if kernels[0].HasVirtio {
		t.Error("expected HasVirtio=false for Xen-only kernel")
	}
	if !kernels[0].IsXenPV {
		t.Error("expected IsXenPV=true when only xen-blkfront present")
	}
}
