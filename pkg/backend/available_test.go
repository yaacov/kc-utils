//go:build unix

package backend

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestAvailableDirectRequiresRoot(t *testing.T) {
	prev := Probes
	t.Cleanup(func() { Probes = prev })

	Probes.OnLinux = func() bool { return true }
	Probes.HasRoot = func() bool { return false }
	Probes.HasKVM = func() bool { return true }

	p := &stubPlugin{name: NameDirect, req: Requirements{Linux: true, Root: true}}
	if p.Available() {
		t.Fatal("expected direct unavailable without root")
	}

	Probes.HasRoot = func() bool { return true }
	if !p.Available() {
		t.Fatal("expected direct available with root")
	}
}

func TestAvailableGuestfsRequiresKVM(t *testing.T) {
	prev := Probes
	t.Cleanup(func() { Probes = prev })

	Probes.OnLinux = func() bool { return true }
	Probes.HasRoot = func() bool { return true }
	Probes.HasKVM = func() bool { return false }
	Probes.HasGuestfish = func() bool { return true }

	p := &stubPlugin{name: NameGuestfs, req: Requirements{Linux: true, KVM: true, Guestfish: true}}
	if p.Available() {
		t.Fatal("expected guestfs unavailable without kvm")
	}

	Probes.HasKVM = func() bool { return true }
	if !p.Available() {
		t.Fatal("expected guestfs available with kvm")
	}
}

func TestAvailableGuestfsRequiresGuestfish(t *testing.T) {
	prev := Probes
	t.Cleanup(func() { Probes = prev })

	Probes.OnLinux = func() bool { return true }
	Probes.HasRoot = func() bool { return true }
	Probes.HasKVM = func() bool { return true }
	Probes.HasGuestfish = func() bool { return false }

	p := &stubPlugin{name: NameGuestfs, req: Requirements{Linux: true, KVM: true, Guestfish: true}}
	if p.Available() {
		t.Fatal("expected guestfs unavailable without guestfish")
	}

	Probes.HasGuestfish = func() bool { return true }
	if !p.Available() {
		t.Fatal("expected guestfs available with guestfish")
	}
}

type stubPlugin struct {
	name string
	req  Requirements
}

func (s *stubPlugin) Name() string               { return s.name }
func (s *stubPlugin) Requirements() Requirements { return s.req }
func (s *stubPlugin) Available() bool            { return Available(s) }
func (s *stubPlugin) New() Backend               { return nil }
func (s *stubPlugin) NewMounted([]types.DiskSpec, string, []types.DiskInfo) (Backend, error) {
	return nil, nil
}
func (s *stubPlugin) TeardownMountRoot(string) error { return nil }
