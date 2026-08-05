//go:build linux

package bcdeditor

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestConvertToVirtio_CopiesFallback(t *testing.T) {
	guestRoot := t.TempDir()
	espPath := "boot/efi"

	msBootDir := filepath.Join(guestRoot, espPath, "EFI", "Microsoft", "Boot")
	if err := os.MkdirAll(msBootDir, 0o755); err != nil {
		t.Fatal(err)
	}

	srcContent := []byte("fake-bootmgfw-binary")
	if err := os.WriteFile(filepath.Join(msBootDir, "bootmgfw.efi"), srcContent, 0o644); err != nil {
		t.Fatal(err)
	}

	b := &BCDEditor{}
	if err := b.ConvertToVirtio(guestRoot, espPath); err != nil {
		t.Fatalf("ConvertToVirtio returned error: %v", err)
	}

	fallbackPath := filepath.Join(guestRoot, espPath, "EFI", "Boot", "bootx64.efi")
	data, err := os.ReadFile(fallbackPath)
	if err != nil {
		t.Fatalf("fallback bootloader not created: %v", err)
	}
	if !bytes.Equal(data, srcContent) {
		t.Errorf("fallback content = %q, want %q", data, srcContent)
	}
}

func TestConvertToVirtio_SkipsExistingFallback(t *testing.T) {
	guestRoot := t.TempDir()
	espPath := "boot/efi"

	msBootDir := filepath.Join(guestRoot, espPath, "EFI", "Microsoft", "Boot")
	if err := os.MkdirAll(msBootDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(msBootDir, "bootmgfw.efi"), []byte("src"), 0o644); err != nil {
		t.Fatal(err)
	}

	fallbackDir := filepath.Join(guestRoot, espPath, "EFI", "Boot")
	if err := os.MkdirAll(fallbackDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fallbackDir, "bootx64.efi"), []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	b := &BCDEditor{}
	if err := b.ConvertToVirtio(guestRoot, espPath); err != nil {
		t.Fatalf("ConvertToVirtio returned error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(fallbackDir, "bootx64.efi"))
	if string(data) != "existing" {
		t.Errorf("existing fallback was overwritten, got %q", data)
	}
}

func TestConvertToVirtio_NoBootloader(t *testing.T) {
	guestRoot := t.TempDir()
	espPath := "boot/efi"

	b := &BCDEditor{}
	if err := b.ConvertToVirtio(guestRoot, espPath); err != nil {
		t.Fatalf("ConvertToVirtio returned error: %v", err)
	}
}
