package guestcaps

import (
	"log/slog"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

// Build fills GuestCaps from inspection and kernel selection results.
func Build(caps *types.GuestCaps, selectedKernel *types.KernelInfo) {
	caps.BlockBus = "virtio"
	caps.NetBus = "virtio"
	caps.VirtioRNG = true
	caps.VirtioBalloon = true
	caps.VirtioSocket = true
	caps.ISAPVPanic = true
	caps.Virtio10 = true
	caps.RTCUTC = true

	// Keep historical virtio defaults when kernel evidence is unavailable,
	// but downgrade when we positively know the selected kernel lacks support.
	if selectedKernel != nil && !selectedKernel.HasVirtio {
		caps.BlockBus = "ide"
		caps.NetBus = "e1000"
		caps.VirtioRNG = false
		caps.VirtioBalloon = false
		caps.VirtioSocket = false
		caps.ISAPVPanic = false
		caps.Virtio10 = false
	}

	switch caps.Arch {
	case "x86_64":
		caps.MachineType = "q35"
	case "aarch64":
		caps.MachineType = "virt"
	case "ppc64le":
		caps.MachineType = "pseries"
	case "s390x":
		caps.MachineType = "s390-ccw-virtio"
	default:
		slog.Warn("no machine type mapping for architecture, defaulting to q35", "arch", caps.Arch)
		caps.MachineType = "q35"
	}
}
