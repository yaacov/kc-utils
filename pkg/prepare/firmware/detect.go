//go:build linux

package firmware

import "github.com/yaacov/kc-utils/pkg/common/types"

// Detect determines firmware type from disk layout.
func Detect(disks []types.DiskInfo) types.FirmwareInfo {
	fw := types.FirmwareInfo{Type: string(types.FirmwareBIOS)}
	if det, ok := Detectors.Get("gpt-esp"); ok {
		if detected, err := det.Detect(disks); err == nil {
			fw = *detected
		}
	}
	return fw
}
