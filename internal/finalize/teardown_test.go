//go:build linux

package finalize

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTeardownOnlyMountRootFallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "leftover"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{
		MountRoot:  dir,
		UseGuestfs: true,
	}
	if err := TeardownOnly(cfg); err != nil {
		t.Fatalf("TeardownOnly: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected cleared mount root, got %v", entries)
	}
}
