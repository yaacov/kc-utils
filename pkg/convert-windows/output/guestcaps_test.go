//go:build linux

package output

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestBuildFallsBackWithoutVirtioDrivers(t *testing.T) {
	caps := &types.GuestCaps{Arch: "x86_64"}
	Build(caps, nil)

	if caps.BlockBus != "ide" {
		t.Errorf("BlockBus = %q, want ide", caps.BlockBus)
	}
	if caps.NetBus != "e1000" {
		t.Errorf("NetBus = %q, want e1000", caps.NetBus)
	}
	if caps.VirtioRNG || caps.VirtioBalloon || caps.VirtioSocket || caps.Virtio10 {
		t.Error("virtio feature flags should be false without copied drivers")
	}
	if !caps.ISAPVPanic {
		t.Error("ISAPVPanic should remain true")
	}
	if caps.MachineType != "q35" {
		t.Errorf("MachineType = %q, want q35", caps.MachineType)
	}
}

func TestBuildUsesCopiedDrivers(t *testing.T) {
	caps := &types.GuestCaps{Arch: "aarch64"}
	Build(caps, []string{"vioscsi", "netkvm", "viorng", "balloon", "viosock", "pvpanic"})

	if caps.BlockBus != "virtio" || caps.NetBus != "virtio" {
		t.Errorf("unexpected buses: block=%q net=%q", caps.BlockBus, caps.NetBus)
	}
	if !caps.VirtioRNG || !caps.VirtioBalloon || !caps.VirtioSocket || !caps.ISAPVPanic || !caps.Virtio10 {
		t.Error("virtio feature flags should be true when matching drivers are present")
	}
	if caps.MachineType != "virt" {
		t.Errorf("MachineType = %q, want virt", caps.MachineType)
	}
}
