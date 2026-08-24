//go:build unix

package windowsvol

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/yaacov/kc-utils/pkg/backend"
	"github.com/yaacov/kc-utils/pkg/common/types"
)

// GPT partition type GUIDs for Windows volume layouts.
const (
	gptLDMMetadata             = "5808c8aa-7e8f-42e0-85d2-e1e90434cfb3"
	gptLDMData                 = "af9b60a0-1431-4f62-bc68-3311714a69ad"
	gptStorageSpacesProtective = "e75caf8f-f680-4cee-afa3-b001e56ef2da"
	gptStorageSpacesData       = "90442c87-50cb-4b44-9a9e-5bd169bd69e7"
	mbrLDMMetadata             = "0x42"
)

// Kind identifies a Windows-specific volume layout that needs special handling.
type Kind string

const (
	KindLDM           Kind = "ldm"
	KindBitLocker     Kind = "bitlocker"
	KindStorageSpaces Kind = "storage_spaces"
)

// Issue is one detected Windows volume feature on a block device.
type Issue struct {
	Kind   Kind
	Device string
}

// PartProbe returns partition type and filesystem type for a device.
type PartProbe func(device string) (partType, fsType string)

// ScanDiskInfos inspects discovered partitions for LDM metadata, BitLocker, or
// Storage Spaces signatures.
func ScanDiskInfos(disks []types.DiskInfo, probe PartProbe) []Issue {
	var issues []Issue
	seen := make(map[string]bool)
	for _, d := range disks {
		for _, p := range d.Partitions {
			if p.DevicePath == "" || seen[p.DevicePath] {
				continue
			}
			seen[p.DevicePath] = true
			fsType := p.FSType
			partType := ""
			if probe != nil {
				pt, ft := probe(p.DevicePath)
				if partType == "" {
					partType = pt
				}
				if fsType == "" {
					fsType = ft
				}
			}
			if kind, ok := Classify(partType, fsType); ok {
				issues = append(issues, Issue{Kind: kind, Device: p.DevicePath})
			}
		}
	}
	return issues
}

// Classify maps partition/filesystem metadata to a Windows volume kind.
func Classify(partType, fsType string) (Kind, bool) {
	fsType = strings.ToLower(strings.TrimSpace(fsType))
	partType = strings.ToLower(strings.TrimSpace(partType))

	if fsType == "bitlocker" {
		return KindBitLocker, true
	}
	if partType == gptLDMMetadata || partType == gptLDMData || partType == mbrLDMMetadata ||
		strings.Contains(partType, "5808c8aa") || strings.Contains(partType, "af9b60a0") {
		return KindLDM, true
	}
	if partType == gptStorageSpacesProtective || partType == gptStorageSpacesData ||
		strings.Contains(partType, "e75caf8f") || strings.Contains(partType, "90442c87") {
		return KindStorageSpaces, true
	}
	return "", false
}

// UnsupportedError is returned when direct/guestfs encounter a volume layout
// that only the qemu backend can handle offline.
func UnsupportedError(kind Kind, device, currentBackend string) error {
	return fmt.Errorf("windows volume %s on %s requires backend %q (current: %q)",
		kind, device, backend.NameQEMU, currentBackend)
}

// FirstUnsupported returns the first issue that the named backend cannot handle.
func FirstUnsupported(issues []Issue, backendName string) *Issue {
	if backendName == backend.NameQEMU {
		for i := range issues {
			if issues[i].Kind == KindStorageSpaces {
				return &issues[i]
			}
		}
		return nil
	}
	for i := range issues {
		switch issues[i].Kind {
		case KindLDM, KindBitLocker, KindStorageSpaces:
			return &issues[i]
		}
	}
	return nil
}

// HostPartProbe uses lsblk and blkid on the conversion host (direct backend).
func HostPartProbe(device string) (partType, fsType string) {
	if out, err := exec.Command("lsblk", "-no", "PARTTYPE", device).Output(); err == nil {
		partType = strings.TrimSpace(string(out))
	}
	if out, err := exec.Command("blkid", "-o", "value", "-s", "TYPE", device).Output(); err == nil {
		fsType = strings.TrimSpace(string(out))
	}
	return partType, fsType
}
