//go:build linux

package backend_test

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/backend"

	_ "github.com/yaacov/kc-utils/pkg/backend/plugins/guestfs"
)

func TestResolveGuestfsWithoutGuestfish(t *testing.T) {
	prev := backend.Probes
	t.Cleanup(func() { backend.Probes = prev })

	backend.Probes.OnLinux = func() bool { return true }
	backend.Probes.HasRoot = func() bool { return true }
	backend.Probes.HasKVM = func() bool { return true }
	backend.Probes.HasGuestfish = func() bool { return false }

	_, err := backend.Resolve(backend.NameGuestfs)
	if err == nil {
		t.Fatal("expected guestfs unavailable without guestfish in PATH")
	}
}
