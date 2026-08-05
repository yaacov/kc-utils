package gptesp

import (
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/prepare/firmware"
)

type GPTESPDetector struct{}

func init() {
	firmware.Detectors.Register("gpt-esp", &GPTESPDetector{})
}

func (g *GPTESPDetector) Detect(disks []types.DiskInfo) (*types.FirmwareInfo, error) {
	for _, d := range disks {
		for _, p := range d.Partitions {
			if !looksLikeESP(p) {
				continue
			}
			if p.FSType == "vfat" {
				return &types.FirmwareInfo{
					Type:       string(types.FirmwareUEFI),
					ESPDevices: []string{p.DevicePath},
				}, nil
			}
		}
	}
	return &types.FirmwareInfo{Type: string(types.FirmwareBIOS)}, nil
}

func looksLikeESP(p types.PartitionInfo) bool {
	if p.FSType != "vfat" {
		return false
	}
	switch p.MountPoint {
	case "/boot/efi", "/efi":
		return true
	}
	if p.SizeBytes > 0 && p.SizeBytes <= 1024*1024*1024 && p.Index == 1 {
		return true
	}
	if strings.Contains(strings.ToLower(p.DevicePath), "efi") {
		return true
	}
	return false
}
