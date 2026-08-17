//go:build linux

package finalize

import (
	"os"
	"path/filepath"
	"testing"

	_ "github.com/yaacov/kc-utils/pkg/guest/plugins/direct"
	_ "github.com/yaacov/kc-utils/pkg/guest/plugins/guestfs"
)

func TestTeardownOnlyMountRootFallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "leftover"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{
		MountRoot: dir,
		Backend:   "guestfs",
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
