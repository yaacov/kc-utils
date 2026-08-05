//go:build linux

package rpm

import (
	"os"
	"path/filepath"
	"testing"
)

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, path string, data []byte, perm os.FileMode) {
	t.Helper()
	mustMkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, data, perm); err != nil {
		t.Fatal(err)
	}
}

func TestScanKernelsSorted(t *testing.T) {
	root := t.TempDir()
	for _, ver := range []string{"5.14.0-100", "5.14.0-284", "4.18.0-305"} {
		mustMkdirAll(t, filepath.Join(root, "lib", "modules", ver))
		mustWriteFile(t, filepath.Join(root, "boot", "vmlinuz-"+ver), nil, 0o644)
		mustWriteFile(t, filepath.Join(root, "boot", "initramfs-"+ver+".img"), nil, 0o644)
	}

	s := &Scanner{}
	kernels, err := s.ScanKernels(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(kernels) != 3 {
		t.Fatalf("got %d kernels, want 3", len(kernels))
	}
	if kernels[0].Version != "5.14.0-284" {
		t.Errorf("first kernel = %q, want 5.14.0-284", kernels[0].Version)
	}
	if kernels[2].Version != "4.18.0-305" {
		t.Errorf("last kernel = %q, want 4.18.0-305", kernels[2].Version)
	}
}

func TestScanKernelsPaths(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "lib", "modules", "5.14.0-284"))
	mustWriteFile(t, filepath.Join(root, "boot", "vmlinuz-5.14.0-284"), nil, 0o644)
	mustWriteFile(t, filepath.Join(root, "boot", "initramfs-5.14.0-284.img"), nil, 0o644)

	s := &Scanner{}
	kernels, err := s.ScanKernels(root)
	if err != nil {
		t.Fatal(err)
	}
	if kernels[0].Path != "/boot/vmlinuz-5.14.0-284" {
		t.Errorf("Path = %q, want /boot/vmlinuz-5.14.0-284", kernels[0].Path)
	}
	if kernels[0].InitrdPath != "/boot/initramfs-5.14.0-284.img" {
		t.Errorf("InitrdPath = %q, want /boot/initramfs-5.14.0-284.img", kernels[0].InitrdPath)
	}
}

func TestFindVmlinuxFallback(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "lib", "modules", "5.14.0-284"))
	mustWriteFile(t, filepath.Join(root, "boot", "vmlinux-5.14.0-284"), nil, 0o644)

	s := &Scanner{}
	kernels, err := s.ScanKernels(root)
	if err != nil {
		t.Fatal(err)
	}
	if kernels[0].Path != "/boot/vmlinux-5.14.0-284" {
		t.Errorf("Path = %q, want /boot/vmlinux-5.14.0-284", kernels[0].Path)
	}
}

func TestFindInitrdFallbacks(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "lib", "modules", "5.14.0-284"))
	mustWriteFile(t, filepath.Join(root, "boot", "initrd.img-5.14.0-284"), nil, 0o644)

	s := &Scanner{}
	kernels, err := s.ScanKernels(root)
	if err != nil {
		t.Fatal(err)
	}
	if kernels[0].InitrdPath != "/boot/initrd.img-5.14.0-284" {
		t.Errorf("InitrdPath = %q, want /boot/initrd.img-5.14.0-284", kernels[0].InitrdPath)
	}
}

func TestSelectBestRequiresBoth(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "lib", "modules", "5.14.0-284"))
	mustWriteFile(t, filepath.Join(root, "boot", "vmlinuz-5.14.0-284"), nil, 0o644)
	mustMkdirAll(t, filepath.Join(root, "lib", "modules", "4.18.0-305"))
	mustWriteFile(t, filepath.Join(root, "boot", "vmlinuz-4.18.0-305"), nil, 0o644)
	mustWriteFile(t, filepath.Join(root, "boot", "initramfs-4.18.0-305.img"), nil, 0o644)

	s := &Scanner{}
	kernels, err := s.ScanKernels(root)
	if err != nil {
		t.Fatal(err)
	}
	best := s.SelectBest(kernels)
	if best == nil {
		t.Fatal("SelectBest returned nil")
	}
	if best.Version != "4.18.0-305" {
		t.Errorf("SelectBest = %q, want 4.18.0-305 (has both path and initrd)", best.Version)
	}
}

func TestSelectBestFallbackFirst(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "lib", "modules", "5.14.0-284"))

	s := &Scanner{}
	kernels, err := s.ScanKernels(root)
	if err != nil {
		t.Fatal(err)
	}
	best := s.SelectBest(kernels)
	if best == nil {
		t.Fatal("SelectBest returned nil")
	}
	if best.Version != "5.14.0-284" {
		t.Errorf("SelectBest fallback = %q, want 5.14.0-284", best.Version)
	}
}

func TestSelectBestEmpty(t *testing.T) {
	s := &Scanner{}
	if s.SelectBest(nil) != nil {
		t.Error("SelectBest(nil) should return nil")
	}
}

func TestScanKernelsEmptyDir(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "lib", "modules"))

	s := &Scanner{}
	kernels, err := s.ScanKernels(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(kernels) != 0 {
		t.Errorf("got %d kernels, want 0", len(kernels))
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
	ver := "5.14.0-284"
	mustMkdirAll(t, filepath.Join(root, "lib", "modules", ver))
	mustWriteFile(t, filepath.Join(root, "boot", "vmlinuz-"+ver), nil, 0o644)
	mustWriteFile(t, filepath.Join(root, "boot", "initramfs-"+ver+".img"), nil, 0o644)
	mustMkdirAll(t, filepath.Join(root, "lib", "modules", ver, "kernel", "drivers", "virtio"))
	mustWriteFile(t, filepath.Join(root, "lib", "modules", ver, "kernel", "drivers", "virtio", "virtio_pci.ko"), nil, 0o644)

	s := &Scanner{}
	kernels, err := s.ScanKernels(root)
	if err != nil {
		t.Fatal(err)
	}
	if !kernels[0].HasVirtio {
		t.Error("expected HasVirtio=true when virtio_pci.ko is present")
	}
	if kernels[0].IsXenPV {
		t.Error("expected IsXenPV=false")
	}
}

func TestScanKernelsIsXenPV(t *testing.T) {
	root := t.TempDir()
	ver := "4.18.0-100"
	mustMkdirAll(t, filepath.Join(root, "lib", "modules", ver))
	mustWriteFile(t, filepath.Join(root, "boot", "vmlinuz-"+ver), nil, 0o644)
	mustWriteFile(t, filepath.Join(root, "boot", "initramfs-"+ver+".img"), nil, 0o644)
	mustMkdirAll(t, filepath.Join(root, "lib", "modules", ver, "kernel", "drivers", "block"))
	mustWriteFile(t, filepath.Join(root, "lib", "modules", ver, "kernel", "drivers", "block", "xen-blkfront.ko"), nil, 0o644)

	s := &Scanner{}
	kernels, err := s.ScanKernels(root)
	if err != nil {
		t.Fatal(err)
	}
	if kernels[0].HasVirtio {
		t.Error("expected HasVirtio=false for Xen-only kernel")
	}
	if !kernels[0].IsXenPV {
		t.Error("expected IsXenPV=true when only xen-blkfront is present")
	}
}

func TestScanKernelsSkipsProbeWithoutVmlinuz(t *testing.T) {
	root := t.TempDir()
	ver := "5.14.0-421.el9.x86_64"
	mustMkdirAll(t, filepath.Join(root, "lib", "modules", ver, "kernel", "drivers", "virtio"))
	mustWriteFile(t, filepath.Join(root, "lib", "modules", ver, "kernel", "drivers", "virtio", "virtio_pci.ko"), nil, 0o644)

	s := &Scanner{}
	kernels, err := s.ScanKernels(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(kernels) != 1 {
		t.Fatalf("got %d kernels, want 1", len(kernels))
	}
	if kernels[0].Path != "" {
		t.Errorf("Path = %q, want empty (no vmlinuz)", kernels[0].Path)
	}
	if kernels[0].HasVirtio {
		t.Error("expected HasVirtio=false when vmlinuz is missing (probe should be skipped)")
	}
}

func TestScanKernelsNoVirtioModules(t *testing.T) {
	root := t.TempDir()
	ver := "5.14.0-284"
	mustMkdirAll(t, filepath.Join(root, "lib", "modules", ver))
	mustWriteFile(t, filepath.Join(root, "boot", "vmlinuz-"+ver), nil, 0o644)

	s := &Scanner{}
	kernels, err := s.ScanKernels(root)
	if err != nil {
		t.Fatal(err)
	}
	if kernels[0].HasVirtio {
		t.Error("expected HasVirtio=false when no virtio modules")
	}
	if kernels[0].IsXenPV {
		t.Error("expected IsXenPV=false when no xen modules")
	}
}
