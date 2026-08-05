//go:build linux

package guest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestTeardownDiscardGuestfsNoopOnHostTree(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "host-only")
	if err := os.WriteFile(marker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	g, err := AttachMounted(
		[]types.DiskSpec{{Path: "/nonexistent.img", Format: "raw"}},
		dir,
		ModeGuestfs,
		[]types.DiskInfo{{Path: "/nonexistent.img", Format: "raw"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.TeardownDiscard(); err != nil {
		t.Fatalf("TeardownDiscard: %v", err)
	}
	// Guestfs does not populate mountRoot; teardown does not wipe it.
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("expected host marker kept: %v", err)
	}
}

func TestTeardownGuestfsSyncNoop(t *testing.T) {
	dir := t.TempDir()
	g, err := AttachMounted(
		[]types.DiskSpec{{Path: "/nonexistent.img", Format: "raw"}},
		dir,
		ModeGuestfs,
		[]types.DiskInfo{{Path: "/nonexistent.img", Format: "raw"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Sync(); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if err := g.Teardown(); err != nil {
		t.Fatalf("Teardown: %v", err)
	}
}

func TestTeardownMountRootGuestfs(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := TeardownMountRoot(dir, ModeGuestfs); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty dir, got %v", entries)
	}
}
