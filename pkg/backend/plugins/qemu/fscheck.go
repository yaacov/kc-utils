//go:build unix

package qemu

import (
	"fmt"
	"strings"
)

// FSType detects the filesystem type of a device via blkid.
func (b *Backend) FSType(device string) (string, error) {
	return b.BlkidAttr(device, "TYPE")
}

// BlkidAttr returns a blkid attribute (TYPE, UUID, LABEL) for a device.
func (b *Backend) BlkidAttr(device, attr string) (string, error) {
	out, err := b.session.client.run("blkid", "-o", "value", "-s", strings.ToUpper(attr), device)
	if err != nil {
		return "", err
	}
	val := strings.TrimSpace(string(out))
	if val == "" {
		return "", fmt.Errorf("blkid %s %s: empty", attr, device)
	}
	return val, nil
}

// FSCheck runs the filesystem checker appropriate for fs on device. Unknown
// filesystem types are skipped (no error), matching the direct backend.
func (b *Backend) FSCheck(device, fs string) error {
	argv := fsckArgv(device, fs)
	if argv == nil {
		return nil
	}
	if _, err := b.session.client.run(argv...); err != nil {
		return fmt.Errorf("fscheck %s: %w", device, err)
	}
	return nil
}

// fsckArgv maps a filesystem type to its non-interactive checker command, or nil
// for an unknown type (which is skipped). Pure, so the mapping is unit-testable.
func fsckArgv(device, fs string) []string {
	switch strings.ToLower(fs) {
	case "ext4", "ext3", "ext2":
		return []string{"e2fsck", "-f", "-y", device}
	case "xfs":
		return []string{"xfs_repair", device}
	case "btrfs":
		return []string{"btrfs", "check", device}
	case "ntfs", "ntfs3":
		return []string{"ntfsfix", "-d", device}
	default:
		return nil
	}
}
