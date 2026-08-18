//go:build unix

package grubfallback

import (
	"os"
	"path/filepath"
	"testing"
)

func mkdirWrite(t *testing.T, dir, file string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, file), content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestConvertToVirtioCreatesBootFallback(t *testing.T) {
	root := t.TempDir()
	efiDir := filepath.Join(root, "boot", "efi", "EFI")
	mkdirWrite(t, filepath.Join(efiDir, "redhat"), "shimx64.efi", []byte("shim-binary"))

	g := &GrubFallback{}
	if err := g.ConvertToVirtio(root, "boot/efi"); err != nil {
		t.Fatalf("ConvertToVirtio error: %v", err)
	}

	fallback := filepath.Join(efiDir, "BOOT", "bootx64.efi")
	if _, err := os.Stat(fallback); err != nil {
		t.Fatalf("fallback bootloader not created: %v", err)
	}

	data, _ := os.ReadFile(fallback)
	if string(data) != "shim-binary" {
		t.Errorf("fallback content = %q, want shim-binary", data)
	}
}

func TestConvertToVirtioSkipsExisting(t *testing.T) {
	root := t.TempDir()
	efiDir := filepath.Join(root, "boot", "efi", "EFI")
	mkdirWrite(t, filepath.Join(efiDir, "BOOT"), "bootx64.efi", []byte("existing"))
	mkdirWrite(t, filepath.Join(efiDir, "redhat"), "shimx64.efi", []byte("new-shim"))

	g := &GrubFallback{}
	if err := g.ConvertToVirtio(root, "boot/efi"); err != nil {
		t.Fatalf("ConvertToVirtio error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(efiDir, "BOOT", "bootx64.efi"))
	if string(data) != "existing" {
		t.Error("existing fallback should not be overwritten")
	}
}

func TestConvertToVirtioNoShimNoOp(t *testing.T) {
	root := t.TempDir()
	efiDir := filepath.Join(root, "boot", "efi", "EFI")
	if err := os.MkdirAll(efiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	g := &GrubFallback{}
	if err := g.ConvertToVirtio(root, "boot/efi"); err != nil {
		t.Fatalf("ConvertToVirtio error: %v", err)
	}

	bootDir := filepath.Join(efiDir, "BOOT")
	if _, err := os.Stat(bootDir); !os.IsNotExist(err) {
		t.Error("BOOT dir should not be created when no shim exists")
	}
}
