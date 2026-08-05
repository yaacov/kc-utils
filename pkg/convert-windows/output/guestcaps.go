package output

import (
	"log/slog"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

// Build fills GuestCaps from copied driver names when available.
func Build(caps *types.GuestCaps, copiedDrivers []string) {
	has := func(name string) bool {
		for _, d := range copiedDrivers {
			if strings.EqualFold(d, name) {
				return true
			}
		}
		return false
	}

	if has("viostor") || has("vioscsi") {
		caps.BlockBus = "virtio"
	} else {
		caps.BlockBus = "ide"
	}
	if has("netkvm") {
		caps.NetBus = "virtio"
	} else {
		caps.NetBus = "e1000"
	}
	caps.VirtioRNG = has("viorng")
	caps.VirtioBalloon = has("balloon")
	caps.VirtioSocket = has("vioserial") || has("viosock")
	caps.ISAPVPanic = true
	caps.Virtio10 = caps.BlockBus == "virtio" || caps.NetBus == "virtio"

	switch caps.Arch {
	case "x86_64":
		caps.MachineType = "q35"
	case "aarch64":
		caps.MachineType = "virt"
	default:
		slog.Warn("no machine type mapping for architecture, defaulting to q35", "arch", caps.Arch)
		caps.MachineType = "q35"
	}
}
