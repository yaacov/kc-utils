//go:build linux

package guestio

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/guest"
	"github.com/yaacov/kc-utils/pkg/guest/backend"

	_ "github.com/yaacov/kc-utils/pkg/guest/plugins/direct"
)

// activeGuestAt attaches a direct guest rooted at root and registers it as the
// active handle, cleaned up when the test ends.
func activeGuestAt(t *testing.T, root string) *guest.Guest {
	t.Helper()
	g, err := guest.AttachMounted(nil, root, backend.ModeDirect, nil)
	if err != nil {
		t.Fatalf("AttachMounted: %v", err)
	}
	guest.SetActive(g)
	t.Cleanup(guest.ClearActive)
	return g
}

func TestGuestPathFromHostNoActive(t *testing.T) {
	guest.ClearActive()
	_, _, ok := guestPathFromHost("/some/path")
	if ok {
		t.Fatal("expected ok=false with no active guest")
	}
}

func TestGuestPathFromHostOutsideRoot(t *testing.T) {
	activeGuestAt(t, t.TempDir())
	_, _, ok := guestPathFromHost("/tmp/outside-root-xyz")
	if ok {
		t.Fatal("expected ok=false for path outside guest root")
	}
}

func TestGuestPathFromHostInsideRoot(t *testing.T) {
	root := t.TempDir()
	g := activeGuestAt(t, root)

	gp, got, ok := guestPathFromHost(root + "/etc/fstab")
	if !ok {
		t.Fatal("expected ok=true for path inside guest root")
	}
	if got != g {
		t.Fatal("expected same guest handle")
	}
	if gp != "/etc/fstab" {
		t.Fatalf("got guest path %q, want /etc/fstab", gp)
	}
}

func TestGuestPathFromHostAtRoot(t *testing.T) {
	root := t.TempDir()
	activeGuestAt(t, root)

	gp, _, ok := guestPathFromHost(root)
	if !ok {
		t.Fatal("expected ok=true for root path")
	}
	if gp != "/" {
		t.Fatalf("got guest path %q, want /", gp)
	}
}
