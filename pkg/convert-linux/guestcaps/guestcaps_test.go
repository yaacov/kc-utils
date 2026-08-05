package guestcaps

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestBuildWithoutVirtioKernelFallsBack(t *testing.T) {
	caps := &types.GuestCaps{Arch: "x86_64"}
	Build(caps, &types.KernelInfo{HasVirtio: false})

	if caps.BlockBus != "ide" {
		t.Errorf("BlockBus = %q, want ide", caps.BlockBus)
	}
	if caps.NetBus != "e1000" {
		t.Errorf("NetBus = %q, want e1000", caps.NetBus)
	}
	if caps.VirtioRNG || caps.VirtioBalloon || caps.VirtioSocket || caps.ISAPVPanic || caps.Virtio10 {
		t.Error("virtio capabilities should be false without virtio kernel support")
	}
	if caps.MachineType != "q35" {
		t.Errorf("MachineType = %q, want q35", caps.MachineType)
	}
}

func TestBuildWithoutKernelEvidenceKeepsVirtioDefaults(t *testing.T) {
	caps := &types.GuestCaps{Arch: "x86_64"}
	Build(caps, nil)

	if caps.BlockBus != "virtio" || caps.NetBus != "virtio" {
		t.Errorf("unexpected buses: block=%q net=%q", caps.BlockBus, caps.NetBus)
	}
	if !caps.VirtioRNG || !caps.VirtioBalloon || !caps.VirtioSocket || !caps.ISAPVPanic || !caps.Virtio10 {
		t.Error("virtio capabilities should remain enabled without kernel evidence")
	}
}

func TestBuildWithVirtioKernelEnablesCapabilities(t *testing.T) {
	caps := &types.GuestCaps{Arch: "aarch64"}
	Build(caps, &types.KernelInfo{HasVirtio: true})

	if caps.BlockBus != "virtio" || caps.NetBus != "virtio" {
		t.Errorf("unexpected buses: block=%q net=%q", caps.BlockBus, caps.NetBus)
	}
	if !caps.VirtioRNG || !caps.VirtioBalloon || !caps.VirtioSocket || !caps.ISAPVPanic || !caps.Virtio10 {
		t.Error("virtio capabilities should be true with virtio kernel support")
	}
	if caps.MachineType != "virt" {
		t.Errorf("MachineType = %q, want virt", caps.MachineType)
	}
}
