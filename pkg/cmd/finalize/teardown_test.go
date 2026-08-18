//go:build linux

package finalize

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/backend"
	"github.com/yaacov/kc-utils/pkg/guest"

	_ "github.com/yaacov/kc-utils/pkg/backend/plugins/direct"
	_ "github.com/yaacov/kc-utils/pkg/backend/plugins/guestfs"
)

func TestTeardownOnlyMountRootFallback(t *testing.T) {
	prev := backend.Probes
	t.Cleanup(func() { backend.Probes = prev })
	backend.Probes.OnLinux = func() bool { return true }
	backend.Probes.HasKVM = func() bool { return true }

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "leftover"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{
		MountRoot: dir,
		Backend:   guest.BackendGuestfs,
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
