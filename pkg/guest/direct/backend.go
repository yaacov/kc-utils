//go:build linux

package direct

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest/direct/disk"
	"github.com/yaacov/kc-utils/pkg/guest/direct/fstype"
	"github.com/yaacov/kc-utils/pkg/guest/direct/luks"
	"github.com/yaacov/kc-utils/pkg/guest/direct/lvm"
	"github.com/yaacov/kc-utils/pkg/guest/direct/mount"
)

// Backend mounts guest disks on the host via mount(8) and requires CAP_SYS_ADMIN.
type Backend struct {
	mountRoot   string
	disks       []*disk.DiskSetup
	mounts      *mount.Manager
	probeMounts *mount.Manager
	diskInfos   []types.DiskInfo
	lvPaths     []string
	diskPaths   []string
	cryptMaps   []string
}

func New() *Backend {
	return &Backend{
		mounts:      mount.NewManager(),
		probeMounts: mount.NewManager(),
	}
}

func NewMounted(disks []types.DiskSpec, mountRoot string, diskInfos []types.DiskInfo) *Backend {
	b := New()
	b.mountRoot = mountRoot
	b.diskInfos = diskInfos
	for _, d := range disks {
		b.diskPaths = append(b.diskPaths, d.Path)
	}
	return b
}

func (b *Backend) Setup(disks []types.DiskSpec, mountRoot string) error {
	b.mountRoot = mountRoot
	for _, d := range disks {
		b.diskPaths = append(b.diskPaths, d.Path)
		slog.Info("setting up disk", "path", d.Path, "format", d.Format)

		ds, err := disk.Setup(d.Path)
		if err != nil {
			_ = b.Teardown()
			return fmt.Errorf("setting up disk %s: %w", d.Path, err)
		}
		b.disks = append(b.disks, ds)

		di := types.DiskInfo{Path: d.Path, Format: d.Format}
		for _, pd := range ds.Partitions {
			ft, _ := fstype.Detect(pd.DevicePath)
			di.Partitions = append(di.Partitions, types.PartitionInfo{
				Index:      pd.Index,
				DevicePath: pd.DevicePath,
				FSType:     ft,
			})
			slog.Info("discovered partition",
				"disk", d.Path,
				"index", pd.Index,
				"device", pd.DevicePath,
				"fstype", ft,
			)
		}
		b.diskInfos = append(b.diskInfos, di)
	}

	var allPartDevices []string
	for _, ds := range b.disks {
		for _, pd := range ds.Partitions {
			allPartDevices = append(allPartDevices, pd.DevicePath)
		}
	}

	if len(allPartDevices) > 0 {
		lvs, err := lvm.ScanAndActivate(allPartDevices)
		if err != nil {
			slog.Warn("LVM scan failed", "error", err)
		} else {
			b.lvPaths = lvs
			slog.Info("LVM activated", "volumes", len(lvs), "paths", lvs)
		}
	}

	return nil
}

func (b *Backend) DiskInfos() []types.DiskInfo { return b.diskInfos }
func (b *Backend) LVPaths() []string           { return b.lvPaths }
func (b *Backend) DiskPaths() []string         { return b.diskPaths }

func (b *Backend) Mount(device, hostMountPoint, ft string, readOnly bool) error {
	if err := os.MkdirAll(hostMountPoint, 0o755); err != nil {
		return err
	}
	return b.mounts.Mount(device, hostMountPoint, ft, readOnly)
}

func (b *Backend) UnmountAll() error {
	return b.mounts.UnmountAll()
}

func (b *Backend) ProbeMount(device, ft, hostMountPoint string) error {
	return b.probeMounts.Mount(device, hostMountPoint, ft, true)
}

func (b *Backend) ProbeUnmount(hostMountPoint string) error {
	return mount.Unmount(hostMountPoint)
}

func (b *Backend) FSType(device string) (string, error) {
	return fstype.Detect(device)
}

func (b *Backend) BlkidAttr(device, attr string) (string, error) {
	out, err := exec.Command("blkid", "-o", "value", "-s", attr, device).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (b *Backend) FSCheck(device, fs string) error {
	switch fs {
	case "ext4", "ext3", "ext2":
		return exec.Command("e2fsck", "-f", "-y", device).Run()
	case "xfs":
		return exec.Command("xfs_repair", device).Run()
	case "btrfs":
		return exec.Command("btrfs", "check", device).Run()
	case "ntfs3":
		return exec.Command("ntfsfix", "-d", device).Run()
	default:
		return nil
	}
}

func (b *Backend) FSTrim(mountpoint string) error {
	return exec.Command("fstrim", "-v", mountpoint).Run()
}

func (b *Backend) Decrypt(device, keyFile, mapperName string) (string, error) {
	mapped, err := luks.Open(device, keyFile, mapperName)
	if err != nil {
		return "", err
	}
	b.cryptMaps = append(b.cryptMaps, mapperName)
	return mapped, nil
}

func (b *Backend) UnlockClevis(device, mapperName string) (string, error) {
	if out, err := exec.Command("clevis", "luks", "list", "-d", device).CombinedOutput(); err != nil {
		return "", fmt.Errorf("clevis pre-flight on %s: %w\n%s", device, err, out)
	}
	if out, err := exec.Command("clevis", "luks", "unlock", "-d", device, "-n", mapperName).CombinedOutput(); err != nil {
		return "", fmt.Errorf("clevis unlock %s: %w\n%s", device, err, out)
	}
	b.cryptMaps = append(b.cryptMaps, mapperName)
	return "/dev/mapper/" + mapperName, nil
}

func (b *Backend) CloseCrypt(mapperName string) error {
	return luks.Close(mapperName)
}

func (b *Backend) RunCommand(guestRoot string, cmd []string) ([]byte, error) {
	return runInGuestRoot(guestRoot, cmd)
}

// runInGuestRoot executes cmd with guestRoot as /. Plain chroot(2) requires
// CAP_SYS_CHROOT; on hosts that deny unprivileged chroot (common on Fedora),
// retry via "unshare -r chroot" which maps root inside a user namespace.
func runInGuestRoot(guestRoot string, cmd []string) ([]byte, error) {
	args := append([]string{guestRoot}, cmd...)
	out, err := exec.Command("chroot", args...).CombinedOutput()
	if err == nil || !chrootNotPermitted(out) {
		return out, err
	}
	unshareArgs := append([]string{"-r", "chroot"}, args...)
	return exec.Command("unshare", unshareArgs...).CombinedOutput()
}

func chrootNotPermitted(out []byte) bool {
	return bytes.Contains(out, []byte("Operation not permitted"))
}

func (b *Backend) DeviceRead(device string, offset int64, size int) ([]byte, error) {
	f, err := os.Open(device)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, size)
	if _, err := f.ReadAt(buf, offset); err != nil {
		return nil, err
	}
	return buf, nil
}

func (b *Backend) DeviceWrite(device string, offset int64, data []byte) error {
	f, err := os.OpenFile(device, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteAt(data, offset)
	return err
}

func (b *Backend) Sync() error {
	return nil
}

func (b *Backend) Release() error {
	if b.probeMounts != nil {
		_ = b.probeMounts.UnmountAll()
	}
	return nil
}

func (b *Backend) Teardown() error {
	return b.teardownResources()
}

// TeardownDiscard is identical to Teardown for the direct backend: writes go
// through live host mounts, so there is nothing extra to discard.
func (b *Backend) TeardownDiscard() error {
	return b.Teardown()
}

// TeardownMountRoot best-effort cleans orphaned direct-backend resources.
func TeardownMountRoot(mountRoot string) error {
	unmountUnder(mountRoot)
	closeAllCryptMaps()
	if err := exec.Command("vgchange", "-an").Run(); err != nil {
		slog.Warn("vgchange -an failed", "error", err)
	}
	return nil
}
