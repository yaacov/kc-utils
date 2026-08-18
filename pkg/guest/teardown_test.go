//go:build unix

package guest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/backend"
	"github.com/yaacov/kc-utils/pkg/common/types"

	_ "github.com/yaacov/kc-utils/pkg/backend/plugins/direct"
	_ "github.com/yaacov/kc-utils/pkg/backend/plugins/guestfs"
)

func enableGuestfsBackendProbes(t *testing.T) {
	t.Helper()
	prev := backend.Probes
	t.Cleanup(func() { backend.Probes = prev })
	backend.Probes.OnLinux = func() bool { return true }
	backend.Probes.HasRoot = func() bool { return true }
	backend.Probes.HasKVM = func() bool { return true }
	backend.Probes.HasGuestfish = func() bool { return true }
}

func TestTeardownDiscardGuestfsNoopOnHostTree(t *testing.T) {
	enableGuestfsBackendProbes(t)
	dir := t.TempDir()
	marker := filepath.Join(dir, "host-only")
	if err := os.WriteFile(marker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	g, err := AttachMounted(
		[]types.DiskSpec{{Path: "/nonexistent.img", Format: "raw"}},
		dir,
		BackendGuestfs,
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
	enableGuestfsBackendProbes(t)
	dir := t.TempDir()
	g, err := AttachMounted(
		[]types.DiskSpec{{Path: "/nonexistent.img", Format: "raw"}},
		dir,
		BackendGuestfs,
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
	enableGuestfsBackendProbes(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := TeardownMountRoot(dir, BackendGuestfs); err != nil {
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

func TestTeardownMountRootGuestfsProbesDisabled(t *testing.T) {
	prev := backend.Probes
	t.Cleanup(func() { backend.Probes = prev })
	backend.Probes.OnLinux = func() bool { return true }
	backend.Probes.HasRoot = func() bool { return true }
	backend.Probes.HasKVM = func() bool { return false }
	backend.Probes.HasGuestfish = func() bool { return false }

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := TeardownMountRoot(dir, BackendGuestfs); err != nil {
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

func TestBackendNameConstants(t *testing.T) {
	if BackendDirect != backend.NameDirect {
		t.Fatalf("BackendDirect=%q want %q", BackendDirect, backend.NameDirect)
	}
	if BackendGuestfs != backend.NameGuestfs {
		t.Fatalf("BackendGuestfs=%q want %q", BackendGuestfs, backend.NameGuestfs)
	}
}
