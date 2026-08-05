//go:build linux

package initramfs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestInjectVirtioModulesNilKernel(t *testing.T) {
	err := InjectVirtioModules("/fake/root", nil)
	if err == nil {
		t.Error("expected error for nil kernel, got nil")
	}
}

func TestInjectVirtioModulesNoVmlinuz(t *testing.T) {
	kernel := &types.KernelInfo{
		Version: "5.14.0-421",
		Path:    "",
	}
	err := InjectVirtioModules("/fake/root", kernel)
	if err == nil {
		t.Error("expected error for kernel without vmlinuz")
	}
}

func TestInferInitrdPathExisting(t *testing.T) {
	guestRoot := t.TempDir()

	bootDir := filepath.Join(guestRoot, "boot")
	if err := os.MkdirAll(bootDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bootDir, "initrd.img-6.1.0-18"), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := inferInitrdPath(guestRoot, "6.1.0-18")
	want := "/boot/initrd.img-6.1.0-18"
	if got != want {
		t.Errorf("inferInitrdPath() = %q, want %q", got, want)
	}
}

func TestInferInitrdPathRPMConvention(t *testing.T) {
	guestRoot := t.TempDir()

	bootDir := filepath.Join(guestRoot, "boot")
	if err := os.MkdirAll(bootDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bootDir, "initramfs-5.14.0-427.img"), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := inferInitrdPath(guestRoot, "5.14.0-427")
	want := "/boot/initramfs-5.14.0-427.img"
	if got != want {
		t.Errorf("inferInitrdPath() = %q, want %q", got, want)
	}
}

func TestInferInitrdPathDefault(t *testing.T) {
	guestRoot := t.TempDir()

	got := inferInitrdPath(guestRoot, "5.14.0-427")
	want := "/boot/initramfs-5.14.0-427.img"
	if got != want {
		t.Errorf("inferInitrdPath() = %q, want %q (should default to RPM convention)", got, want)
	}
}

func TestInjectVirtioModulesEmptyInitrdPath(t *testing.T) {
	kernel := &types.KernelInfo{
		Version:    "5.14.0-427",
		Path:       "/boot/vmlinuz-5.14.0-427",
		InitrdPath: "",
	}

	err := InjectVirtioModules("/nonexistent/root", kernel)
	if err == nil {
		t.Error("expected error (dracut not available), got nil")
	}
}
