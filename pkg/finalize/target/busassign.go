package target

import "github.com/yaacov/kc-utils/pkg/common/types"

// Buses maps source disks to target bus slots.
func Buses(disks []types.DiskInfo, busType string) types.TargetBuses {
	buses := types.TargetBuses{}
	for i := range disks {
		slot := types.BusSlot{Index: i, SourceDisk: i}
		switch busType {
		case "virtio":
			buses.VirtioBlk = append(buses.VirtioBlk, slot)
		case "scsi":
			buses.SCSI = append(buses.SCSI, slot)
		default:
			buses.IDE = append(buses.IDE, slot)
		}
	}
	return buses
}

// NICs builds target NICs from source metadata.
func NICs(sourceNICs []types.NICSpec, netBus string) []types.TargetNIC {
	if len(sourceNICs) == 0 {
		return []types.TargetNIC{{MAC: "00:00:00:00:00:00", Model: netBus}}
	}
	nics := make([]types.TargetNIC, 0, len(sourceNICs))
	for _, src := range sourceNICs {
		nics = append(nics, types.TargetNIC{
			MAC:     src.MAC,
			Model:   netBus,
			Network: src.Network,
		})
	}
	return nics
}
