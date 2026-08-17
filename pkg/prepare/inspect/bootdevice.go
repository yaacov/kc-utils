//go:build unix

package inspect

import (
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

// Detect derives boot device info from mounted disks and the guest root path.
func Detect(mountRoot string, disks []types.DiskInfo) types.BootDeviceInfo {
	info := types.BootDeviceInfo{DiskIndex: 0}

	grub2Cfg := filepath.Join(mountRoot, "boot", "grub2", "grub.cfg")
	grubCfg := filepath.Join(mountRoot, "boot", "grub", "grub.cfg")
	efiDir := filepath.Join(mountRoot, "boot", "efi", "EFI")

	switch {
	case guestio.FileExists(grub2Cfg):
		info.BootloaderType = "grub2"
	case guestio.FileExists(grubCfg):
		info.BootloaderType = "grub"
	case guestio.FileExists(efiDir):
		info.BootloaderType = "uefi"
	}

	for di, d := range disks {
		for _, p := range d.Partitions {
			if p.MountPoint == "/boot" {
				info.DiskIndex = di
				info.PartIndex = p.Index
				return info
			}
		}
	}
	for di, d := range disks {
		for _, p := range d.Partitions {
			if p.MountPoint == "/" {
				info.DiskIndex = di
				info.PartIndex = p.Index
				return info
			}
		}
	}
	return info
}
