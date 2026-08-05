package firmware

import (
	"github.com/yaacov/kc-utils/pkg/common/plugin"
	"github.com/yaacov/kc-utils/pkg/common/types"
)

// FirmwareDetector determines whether the guest uses BIOS or UEFI firmware.
type FirmwareDetector interface {
	Detect(disks []types.DiskInfo) (*types.FirmwareInfo, error)
}

// Detectors is the global registry of FirmwareDetector implementations.
var Detectors = plugin.NewRegistry[string, FirmwareDetector]()
