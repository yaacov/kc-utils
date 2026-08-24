//go:build unix

package qemu

import (
	"fmt"
	"log/slog"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

// Backend boots a minimal appliance with qemu-system-* and drives it through a
// tiny in-guest agent over a unix socket. The appliance exposes only primitive
// operations; all conversion logic (partition/LVM/LUKS discovery, mount
// planning, fs-checks, chroot) is composed here on the host.
type Backend struct {
	session   *vmSession
	mountRoot string

	diskPaths   []string
	diskDevices []string // appliance device per disk index: /dev/vda, /dev/vdb, ...
	diskInfos   []types.DiskInfo
	lvPaths     []string
	ldmPaths    []string
	ldmActive   bool

	mounts      []mountEntry // recorded eager mounts, torn down in reverse
	probeActive string
	cryptMaps   []string
	network     bool
}

// mountEntry records one eager mount for reverse-order teardown.
type mountEntry struct {
	Device        string
	AppliancePath string
}

func New() *Backend {
	return &Backend{network: clevisNetworkRequested()}
}

// NewMounted creates a Backend for a guest already prepared by a prior stage.
// When a shared appliance is advertised (KC_QEMU_SOCK) it is adopted with its
// live mounts intact; otherwise a fresh VM is booted and filesystems remounted
// from the recorded disk info.
func NewMounted(disks []types.DiskSpec, mountRoot string, diskInfos []types.DiskInfo) (*Backend, error) {
	b := New()
	b.mountRoot = mountRoot
	b.diskInfos = diskInfos
	for _, d := range disks {
		b.diskPaths = append(b.diskPaths, d.Path)
	}
	b.assignDiskDevices()

	drives := toDriveSpecs(disks)
	session, adopted, err := adoptVMSession(drives)
	if err != nil {
		return nil, err
	}
	if adopted {
		b.session = session
		if err := b.ensureApplianceRoot(); err != nil {
			return nil, err
		}
		return b, nil
	}

	// No shared appliance: boot a local VM and remount from the recorded plan.
	session, err = newVMSession(drives, b.network)
	if err != nil {
		return nil, err
	}
	b.session = session
	if err := b.ensureApplianceRoot(); err != nil {
		return nil, err
	}
	b.activateLDM()
	if err := b.remountFromDiskInfos(); err != nil {
		return nil, err
	}
	return b, nil
}

func (b *Backend) Setup(disks []types.DiskSpec, mountRoot string) error {
	b.mountRoot = mountRoot
	for _, d := range disks {
		b.diskPaths = append(b.diskPaths, d.Path)
	}
	b.assignDiskDevices()

	if err := b.ensureSession(toDriveSpecs(disks)); err != nil {
		return err
	}
	if err := b.ensureApplianceRoot(); err != nil {
		return err
	}
	slog.Info("qemu backend setup",
		"disks", len(disks),
		"mountRoot", mountRoot,
		"paths", b.diskPaths,
		"shared", b.session.ownedExternally,
	)

	for i, d := range disks {
		di := types.DiskInfo{Path: d.Path, Format: d.Format}
		parts, err := b.discoverPartitions(b.diskDevices[i])
		if err != nil {
			slog.Warn("partition discovery failed", "disk", d.Path, "error", err)
		}
		di.Partitions = parts
		b.diskInfos = append(b.diskInfos, di)
		if len(parts) == 0 {
			slog.Warn("no partitions discovered", "disk", d.Path)
		}
		for _, p := range parts {
			slog.Info("discovered partition",
				"disk", d.Path,
				"index", p.Index,
				"device", p.DevicePath,
				"fstype", p.FSType,
			)
		}
	}

	if err := b.checkWindowsVolumes(); err != nil {
		return err
	}

	b.activateLDM()
	ldmParts, ldmPaths, err := b.discoverLDMVolumes()
	if err != nil {
		slog.Warn("LDM discovery failed", "error", err)
	} else if len(ldmPaths) > 0 {
		slog.Info("LDM volumes discovered", "volumes", len(ldmPaths), "paths", ldmPaths)
		b.mergeLDMIntoDiskInfos(ldmParts)
	}
	b.ldmPaths = ldmPaths

	lvs, err := b.discoverLVs()
	if err != nil {
		slog.Warn("LVM discovery failed", "error", err)
	} else if len(lvs) > 0 {
		slog.Info("LVM activated", "volumes", len(lvs), "paths", lvs)
	}
	b.lvPaths = lvs

	return nil
}

// assignDiskDevices maps each disk index to its appliance virtio-blk device.
// Drives attach in order, so disk i is /dev/vd{a+i}.
func (b *Backend) assignDiskDevices() {
	b.diskDevices = make([]string, len(b.diskPaths))
	for i := range b.diskPaths {
		b.diskDevices[i] = diskDevice(i)
	}
}

// diskDevice returns the virtio-blk device name for the i-th attached disk.
// Linux names virtio-blk disks like SCSI disks: vda..vdz, then vdaa, vdab, …
// (a bijective base-26 sequence), so index 26 is /dev/vdaa, not an out-of-range
// character.
func diskDevice(i int) string {
	name := ""
	for i >= 0 {
		name = string(rune('a'+i%26)) + name
		i = i/26 - 1
	}
	return "/dev/vd" + name
}

// ensureSession boots or adopts an appliance with the given drives.
func (b *Backend) ensureSession(drives []driveSpec) error {
	if b.session != nil && b.session.alive() {
		return nil
	}
	if b.session != nil && !b.session.ownedExternally {
		// Session died; restart and let callers re-mount.
		if err := b.session.restart(); err != nil {
			return err
		}
		return nil
	}
	session, adopted, err := adoptVMSession(drives)
	if err != nil {
		return err
	}
	if adopted {
		b.session = session
		return nil
	}
	session, err = newVMSession(drives, b.network)
	if err != nil {
		return err
	}
	b.session = session
	return nil
}

// ensureApplianceRoot creates the appliance-side mount root.
func (b *Backend) ensureApplianceRoot() error {
	if _, err := b.session.client.run("mkdir", "-p", applianceMountRoot); err != nil {
		return fmt.Errorf("create appliance mount root: %w", err)
	}
	return nil
}

func (b *Backend) DiskInfos() []types.DiskInfo { return b.diskInfos }
func (b *Backend) LVPaths() []string           { return b.lvPaths }
func (b *Backend) LDMPaths() []string          { return b.ldmPaths }
func (b *Backend) DiskPaths() []string         { return b.diskPaths }

// Sync flushes the guest's in-memory writes to the attached virtio-blk devices
// so they reach the backing images. It is a no-op when no session is live.
func (b *Backend) Sync() error {
	if b.session == nil || b.session.client == nil {
		return nil
	}
	if _, err := b.session.client.run("sync"); err != nil {
		return fmt.Errorf("guest sync: %w", err)
	}
	return nil
}

// Release drops the process-local agent connection without tearing down the
// appliance (which must survive for the next stage when shared).
func (b *Backend) Release() error {
	if b.session != nil {
		return b.session.close()
	}
	return nil
}
