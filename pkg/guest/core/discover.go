//go:build unix

package core

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest/runtime"
)

// BlockDevice is a node from `lsblk -J` (a disk, partition, LVM LV, or crypt
// mapping). Children are nested block devices.
type BlockDevice struct {
	Name     string        `json:"name"`
	Path     string        `json:"path"`
	Type     string        `json:"type"`
	FSType   string        `json:"fstype"`
	Serial   string        `json:"serial"`
	Children []BlockDevice `json:"children"`
}

type lsblkOutput struct {
	BlockDevices []BlockDevice `json:"blockdevices"`
}

const lsblkColumns = "NAME,PATH,TYPE,FSTYPE,SERIAL"

// ListBlockDevices returns the top-level block devices (disks) reported by
// lsblk, each with its partition children.
func (b *Backend) ListBlockDevices() ([]BlockDevice, error) {
	res, err := b.run("lsblk", "-J", "-o", lsblkColumns)
	if err != nil {
		return nil, err
	}
	var out lsblkOutput
	if err := json.Unmarshal(res.Stdout, &out); err != nil {
		return nil, fmt.Errorf("parse lsblk output: %w", err)
	}
	disks := make([]BlockDevice, 0, len(out.BlockDevices))
	for _, d := range out.BlockDevices {
		if d.Type == "disk" {
			disks = append(disks, d)
		}
	}
	return disks, nil
}

func (b *Backend) lsblkOne(device string) (BlockDevice, error) {
	res, err := b.run("lsblk", "-J", "-o", lsblkColumns, device)
	if err != nil {
		return BlockDevice{}, err
	}
	var out lsblkOutput
	if err := json.Unmarshal(res.Stdout, &out); err != nil {
		return BlockDevice{}, fmt.Errorf("parse lsblk %s: %w", device, err)
	}
	if len(out.BlockDevices) == 0 {
		return BlockDevice{}, fmt.Errorf("lsblk found no device %s", device)
	}
	return out.BlockDevices[0], nil
}

// DiscoverDevice enumerates the partitions of a single block device via lsblk
// and records a DiskInfo (with imagePath/format as reported by the caller).
// It appends to the backend's disk/partition state for later LVM scanning.
func (b *Backend) DiscoverDevice(device, imagePath, format string) error {
	dev, err := b.lsblkOne(device)
	if err != nil {
		return err
	}
	di := types.DiskInfo{Path: imagePath, Format: format}
	idx := 0
	for _, c := range dev.Children {
		if c.Type != "part" {
			continue
		}
		idx++
		di.Partitions = append(di.Partitions, types.PartitionInfo{
			Index:      idx,
			DevicePath: c.Path,
			FSType:     normalizeFSType(c.FSType),
		})
		b.partDevices = append(b.partDevices, c.Path)
	}
	b.diskInfos = append(b.diskInfos, di)
	b.diskPaths = append(b.diskPaths, imagePath)
	return nil
}

// ScanLVM activates volume groups on the discovered partitions (and any open
// LUKS mappers) and records the resulting logical volume paths.
func (b *Backend) ScanLVM() error {
	lvs, err := b.scanAndActivate(b.lvmDevices())
	if err != nil {
		return err
	}
	b.lvPaths = lvs
	return nil
}

// RescanBlock re-activates LVM after a LUKS unlock so volumes on decrypted
// devices appear in LVPaths.
func (b *Backend) RescanBlock() error { return b.ScanLVM() }

func (b *Backend) lvmDevices() []string {
	devs := append([]string(nil), b.partDevices...)
	for _, m := range b.cryptMaps {
		devs = append(devs, "/dev/mapper/"+m)
	}
	return devs
}

func (b *Backend) scanAndActivate(devices []string) ([]string, error) {
	for _, d := range devices {
		// Best-effort: pvscan exits non-zero for non-PV devices.
		_, _ = b.rt.Run(&runtime.CommandSpec{Argv: []string{"pvscan", "--cache", d}})
	}
	if _, err := b.run("vgscan"); err != nil {
		return nil, err
	}
	if _, err := b.run("vgchange", "-ay"); err != nil {
		return nil, err
	}
	res, err := b.run("lvscan")
	if err != nil {
		return nil, err
	}
	var lvs []string
	for line := range strings.SplitSeq(string(res.Stdout), "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "ACTIVE") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			lvs = append(lvs, strings.Trim(fields[1], "'"))
		}
	}
	return lvs, nil
}

// ---- fstype / blkid ----------------------------------------------------

// FSType reports the filesystem type of device using blkid. It returns "" (not
// an error) when blkid detects no filesystem.
func (b *Backend) FSType(device string) (string, error) {
	res, err := b.rt.Run(&runtime.CommandSpec{Argv: []string{"blkid", "-o", "value", "-s", "TYPE", device}})
	if err != nil {
		return "", fmt.Errorf("blkid %s: %w", device, err)
	}
	return normalizeFSType(strings.TrimSpace(string(res.Stdout))), nil
}

// BlkidAttr returns a single blkid attribute (e.g. UUID) for device, or "".
func (b *Backend) BlkidAttr(device, attr string) (string, error) {
	res, err := b.rt.Run(&runtime.CommandSpec{Argv: []string{"blkid", "-o", "value", "-s", attr, device}})
	if err != nil {
		return "", fmt.Errorf("blkid %s: %w", device, err)
	}
	return strings.TrimSpace(string(res.Stdout)), nil
}

// normalizeFSType maps a blkid/lsblk fstype to the mount type vocabulary used
// across the pipeline (blkid reports "ntfs"; the kernel driver is "ntfs3").
func normalizeFSType(ft string) string {
	ft = strings.TrimSpace(ft)
	if strings.EqualFold(ft, "ntfs") {
		return "ntfs3"
	}
	return ft
}

// readHostFile reads a path that is always on the host (e.g. a LUKS key file
// supplied by the caller), regardless of which runtime is in use.
func readHostFile(path string) ([]byte, error) { return os.ReadFile(path) }
