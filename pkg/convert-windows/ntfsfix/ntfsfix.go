//go:build linux

package ntfsfix

import (
	"encoding/binary"
	"log/slog"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest"
)

// Fix patches the NTFS boot sector number-of-heads field on pre-Vista Windows.
// Thresholds match virt-v2v convert_windows.ml fix_ntfs_heads.
func Fix(majorVersion int, disks []types.DiskInfo) {
	if majorVersion >= 6 {
		return
	}
	if len(disks) == 0 {
		return
	}
	disk := disks[0]
	diskSize := disk.SizeBytes
	heads := headsForSize(diskSize)

	for _, p := range disk.Partitions {
		if p.DevicePath == "" {
			continue
		}
		fs := strings.ToLower(p.FSType)
		if fs != "" && fs != "ntfs" && fs != "ntfs3" {
			continue
		}
		if !patchNTFSHeads(p.DevicePath, heads) {
			continue
		}
		slog.Info("patched NTFS heads", "device", p.DevicePath, "heads", heads)
		return
	}
	slog.Warn("NTFS heads fix: no suitable NTFS partition found on first disk")
}

func patchNTFSHeads(devicePath string, heads uint16) bool {
	g := guest.Active()
	if g == nil {
		slog.Warn("NTFS heads fix: no active guest handle")
		return false
	}
	magic, err := g.DeviceRead(devicePath, 3, 8)
	if err != nil {
		slog.Warn("NTFS heads fix: read magic failed", "device", devicePath, "error", err)
		return false
	}
	if string(magic) != "NTFS    " {
		return false
	}
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], heads)
	if err := g.DeviceWrite(devicePath, 0x1A, buf[:]); err != nil {
		slog.Warn("NTFS heads fix: write failed", "error", err)
		return false
	}
	return true
}

// headsForSize returns virt-v2v's XP/2000 heads table based on whole-disk size.
func headsForSize(sizeBytes int64) uint16 {
	switch {
	case sizeBytes < 2114445312:
		return 0x40
	case sizeBytes < 4228374780:
		return 0x80
	default:
		return 0xFF
	}
}
