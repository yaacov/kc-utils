package fstype

import (
	"fmt"
	"os"
)

// Detect reads superblock magic bytes to identify the filesystem type.
// Returns the kernel module name for use with mount(2).
func Detect(device string) (string, error) {
	f, err := os.Open(device)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", device, err)
	}
	defer f.Close()

	buf := make([]byte, 0x10100)
	n, err := f.Read(buf)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", device, err)
	}
	if n < 512 {
		return "unknown", nil
	}

	return detect(buf[:n]), nil
}

func detect(buf []byte) string {
	// ext2/3/4: magic 0xEF53 at offset 0x438
	if len(buf) > 0x439 && buf[0x438] == 0x53 && buf[0x439] == 0xEF {
		return "ext4"
	}

	// XFS: magic "XFSB" at offset 0
	if len(buf) >= 4 && string(buf[0:4]) == "XFSB" {
		return "xfs"
	}

	// btrfs: magic "_BHRfS_M" at offset 0x10040
	if len(buf) > 0x10047 && string(buf[0x10040:0x10048]) == "_BHRfS_M" {
		return "btrfs"
	}

	// NTFS: magic "NTFS" at offset 3
	if len(buf) >= 7 && string(buf[3:7]) == "NTFS" {
		return "ntfs3"
	}

	// FAT32: check for FAT signature
	if len(buf) >= 0x58 && (string(buf[0x52:0x57]) == "FAT32" || string(buf[0x36:0x3B]) == "FAT16" || string(buf[0x36:0x39]) == "FAT") {
		return "vfat"
	}

	// swap: magic "SWAPSPACE2" or "SWAP-SPACE" at offset 0xFF6 (4K page) or 0x1FF6 (8K page)
	if len(buf) > 0x1000 {
		if string(buf[0xFF6:0x1000]) == "SWAPSPACE2" || string(buf[0xFF6:0x1000]) == "SWAP-SPACE" {
			return "swap"
		}
	}

	return "unknown"
}
