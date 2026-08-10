//go:build linux

package guestfs

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

// Backend uses libguestfs guestfish; no CAP_SYS_ADMIN or /dev/fuse.
type Backend struct {
	diskPaths    []string
	mountRoot    string
	diskInfos    []types.DiskInfo
	lvPaths      []string
	mountSpecs   []gfsMountSpec
	mounted      bool
	mountsActive bool
	inspectDone  bool
	probeActive  string
	session      *guestfishSession
	cryptMaps    []string
}

func New() *Backend {
	return &Backend{}
}

// NewMounted creates a Backend for a guest already prepared by a prior stage.
// It adopts a shared guestfish listener from the environment if available.
func NewMounted(disks []types.DiskSpec, mountRoot string, diskInfos []types.DiskInfo) (*Backend, error) {
	b := New()
	b.mountRoot = mountRoot
	b.mounted = true
	b.diskInfos = diskInfos
	for _, d := range disks {
		b.diskPaths = append(b.diskPaths, d.Path)
	}
	b.mountSpecs = mountSpecsFromDiskInfos(diskInfos)

	if err := b.attachSession(); err != nil {
		return nil, err
	}
	return b, nil
}

func (b *Backend) Setup(disks []types.DiskSpec, mountRoot string) error {
	b.mountRoot = mountRoot

	for _, d := range disks {
		b.diskPaths = append(b.diskPaths, d.Path)
	}

	if err := b.ensureSession(); err != nil {
		return err
	}
	slog.Info("guestfs backend setup",
		"disks", len(disks),
		"mountRoot", mountRoot,
		"LIBGUESTFS_BACKEND", os.Getenv("LIBGUESTFS_BACKEND"),
		"paths", b.diskPaths,
		"guestfishPID", b.session.pid,
		"shared", b.session.ownedExternally,
	)

	for _, d := range disks {
		di := types.DiskInfo{Path: d.Path, Format: d.Format}

		parts, err := b.discoverPartitions(d.Path)
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

	lvs, err := b.discoverLVs()
	if err != nil {
		slog.Warn("LVM discovery failed", "error", err)
	} else if len(lvs) > 0 {
		slog.Info("LVM discovered", "volumes", len(lvs), "paths", lvs)
	}
	b.lvPaths = lvs

	return nil
}

func (b *Backend) ensureSession() error {
	if b.session != nil && b.session.pid > 0 {
		return b.session.ensureLaunched(b.diskPaths)
	}
	s, err := openGuestfishSession()
	if err != nil {
		return err
	}
	b.session = s
	return b.session.ensureLaunched(b.diskPaths)
}

// ensureSessionWithRecovery wraps ensureSession with crash recovery.
// If the session has died, it starts a fresh guestfish listener and
// re-launches the appliance.
func (b *Backend) ensureSessionWithRecovery() error {
	err := b.ensureSession()
	if err == nil {
		return nil
	}
	return b.tryRestart(err)
}

func (b *Backend) tryRestart(origErr error) error {
	if b.session == nil || sessionAlive(b.session.pid) {
		return origErr
	}
	slog.Warn("guestfish session dead, attempting restart", "error", origErr)
	if restartErr := b.session.restart(b.diskPaths); restartErr != nil {
		return fmt.Errorf("session restart failed: %w (original: %v)", restartErr, origErr)
	}
	b.mountsActive = false
	return nil
}

// withRecovery executes fn and, if the session crashed during the call,
// restarts the session, remounts filesystems, and retries once.
func (b *Backend) withRecovery(fn func() ([]byte, error)) ([]byte, error) {
	out, err := fn()
	if err == nil {
		return out, nil
	}
	if b.session == nil || sessionAlive(b.session.pid) {
		return out, err
	}
	slog.Warn("guestfish session died during command, restarting", "error", err)
	if restartErr := b.tryRestart(err); restartErr != nil {
		return nil, restartErr
	}
	if mountErr := b.ensureMounted(); mountErr != nil {
		return nil, mountErr
	}
	return fn()
}

func (b *Backend) attachSession() error {
	if b.session != nil && b.session.pid > 0 {
		return b.session.ensureLaunched(b.diskPaths)
	}
	pid, ok := envGuestfishPID()
	if !ok {
		return nil
	}
	if !sessionAlive(pid) {
		return fmt.Errorf("guestfish shared session pid %d is not running (socket %s)", pid, guestfishSocketPath(pid))
	}
	b.session = &guestfishSession{pid: pid, ownedExternally: true}
	slog.Info("guestfish adopting shared session",
		"pid", pid,
		"guestfishPID", pid,
		"shared", true,
	)
	return b.session.ensureLaunched(b.diskPaths)
}

func (b *Backend) DiskInfos() []types.DiskInfo { return b.diskInfos }
func (b *Backend) LVPaths() []string           { return b.lvPaths }
func (b *Backend) DiskPaths() []string         { return b.diskPaths }

func (b *Backend) Mount(device, hostMountPoint, _ string, _ bool) error {
	guestMount := "/"
	if hostMountPoint != b.mountRoot {
		rel, err := filepath.Rel(b.mountRoot, hostMountPoint)
		if err == nil {
			guestMount = "/" + filepath.ToSlash(rel)
		}
	}

	for _, ms := range b.mountSpecs {
		if ms.GuestMount == guestMount {
			return nil
		}
	}

	b.mountSpecs = append(b.mountSpecs, gfsMountSpec{
		Device:     device,
		GuestMount: guestMount,
	})
	b.mounted = true
	b.mountsActive = false
	return nil
}

func (b *Backend) UnmountAll() error {
	if !b.mounted {
		return nil
	}
	if err := b.ensureSession(); err != nil {
		return err
	}
	if _, err := b.session.remoteScript("-umount-all\n"); err != nil {
		return err
	}
	b.mountSpecs = nil
	b.mounted = false
	b.mountsActive = false
	return nil
}

func (b *Backend) FSType(device string) (string, error) {
	if err := b.ensureSession(); err != nil {
		return "", err
	}
	out, err := b.session.remoteScript("-vfs-type " + quoteGuestfish(device) + "\n")
	if err != nil {
		return "", fmt.Errorf("guestfish vfs-type %s: %w", device, err)
	}
	ft := strings.TrimSpace(string(out))
	if ft == "" {
		return "", fmt.Errorf("guestfish vfs-type %s: empty", device)
	}
	return ft, nil
}

func (b *Backend) BlkidAttr(device, attr string) (string, error) {
	if err := b.ensureSession(); err != nil {
		return "", err
	}
	var gfCmd string
	switch strings.ToUpper(attr) {
	case "UUID":
		gfCmd = "vfs-uuid"
	case "LABEL":
		gfCmd = "vfs-label"
	default:
		return "", fmt.Errorf("unsupported blkid attribute %q", attr)
	}
	out, err := b.session.remoteScript("-" + gfCmd + " " + quoteGuestfish(device) + "\n")
	if err != nil {
		return "", err
	}
	val := strings.TrimSpace(string(out))
	if val == "" {
		return "", fmt.Errorf("guestfish %s %s: empty", gfCmd, device)
	}
	return val, nil
}

// fscheckCommand maps a filesystem type to the guestfish fsck command name.
func fscheckCommand(fstype string) (string, bool) {
	switch strings.ToLower(fstype) {
	case "ext4", "ext3", "ext2":
		return "e2fsck-f", true
	case "xfs":
		return "xfs-repair", true
	case "ntfs", "ntfs3":
		return "ntfsfix", true
	default:
		return "", false
	}
}

func (b *Backend) FSCheck(device, fs string) error {
	if err := b.ensureSession(); err != nil {
		return err
	}
	cmd, ok := fscheckCommand(fs)
	if !ok {
		return nil
	}
	if _, err := b.session.remote(cmd, device); err != nil {
		return fmt.Errorf("guestfish fscheck %s: %w", device, err)
	}
	return nil
}

func (b *Backend) FSTrim(mountpoint string) error {
	if err := b.ensureMounted(); err != nil {
		return err
	}
	guestPath := "/"
	if mountpoint != b.mountRoot {
		if rel, err := filepath.Rel(b.mountRoot, mountpoint); err == nil {
			guestPath = "/" + filepath.ToSlash(rel)
		}
	}
	var script strings.Builder
	script.WriteString("fstrim ")
	script.WriteString(quoteGuestfish(guestPath))
	script.WriteByte('\n')
	if _, err := b.session.remoteScript(script.String()); err != nil {
		return fmt.Errorf("guestfish fstrim: %w", err)
	}
	return nil
}

func (b *Backend) Decrypt(device, keyFile, mapperName string) (string, error) {
	if err := b.ensureSession(); err != nil {
		return "", err
	}
	keyData, err := os.ReadFile(keyFile)
	if err != nil {
		return "", fmt.Errorf("read LUKS keyfile %s: %w", keyFile, err)
	}
	// Passphrases from Forklift /etc/luks files are typically newline-terminated text.
	keyData = bytes.TrimRight(keyData, "\r\n")
	keyData = append(keyData, '\n')

	mapped := "/dev/mapper/" + mapperName
	_, err = runGuestfsCmdWithStdin(keyData, guestfishBinary(),
		b.session.remoteFlag(),
		"--keys-from-stdin",
		"--",
		"cryptsetup-open", device, mapperName,
	)
	if err != nil {
		return "", fmt.Errorf("cryptsetup-open %s: %w", device, err)
	}
	if !b.dmDevicePresent(mapped) {
		return "", fmt.Errorf("cryptsetup-open %s: mapper %s not created", device, mapped)
	}
	b.cryptMaps = append(b.cryptMaps, mapperName)
	slog.Info("guestfs LUKS decrypted with keyfile", "device", device, "mapper", mapped)
	return mapped, nil
}

func (b *Backend) UnlockClevis(device, mapperName string) (string, error) {
	if err := b.ensureSession(); err != nil {
		return "", err
	}
	mapped := "/dev/mapper/" + mapperName
	// Soft command: prepare tries every candidate; non-Clevis devices must not
	// kill the shared listener.
	script := "clevis-luks-unlock " + quoteGuestfish(device) + " " + quoteGuestfish(mapperName) + "\n"
	if _, err := b.session.remoteScriptSoft(script); err != nil {
		return "", fmt.Errorf("clevis-luks-unlock %s: %w", device, err)
	}
	if !b.dmDevicePresent(mapped) {
		return "", fmt.Errorf("clevis-luks-unlock %s: mapper %s not created", device, mapped)
	}
	b.cryptMaps = append(b.cryptMaps, mapperName)
	slog.Info("guestfs Clevis LUKS unlocked", "device", device, "mapper", mapped)
	return mapped, nil
}

func (b *Backend) CloseCrypt(mapperName string) error {
	if err := b.ensureSession(); err != nil {
		return err
	}
	mapped := mapperName
	if !strings.HasPrefix(mapped, "/dev/mapper/") {
		mapped = "/dev/mapper/" + mapperName
	}
	_, err := b.session.remote("cryptsetup-close", mapped)
	return err
}

// RescanBlock runs lvm-scan after LUKS unlock so LVs on unlocked devices appear.
func (b *Backend) RescanBlock() error {
	if err := b.ensureSession(); err != nil {
		return err
	}
	if _, err := b.session.remoteScript("lvm-scan true\n"); err != nil {
		return err
	}
	lvs, err := b.discoverLVs()
	if err != nil {
		return fmt.Errorf("rediscover LVs after decrypt: %w", err)
	}
	b.lvPaths = lvs
	slog.Info("guestfs rescan after decrypt", "lvs", len(lvs), "paths", lvs)
	return nil
}

func (b *Backend) dmDevicePresent(mapped string) bool {
	out, err := b.session.remoteScriptSoft("-list-dm-devices\n")
	if err != nil {
		return false
	}
	return dmOutputContains(string(out), mapped)
}

func dmOutputContains(output, mapped string) bool {
	base := filepath.Base(mapped)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == mapped || filepath.Base(line) == base {
			return true
		}
	}
	return false
}

// mountVirtualFS mounts /proc, /sys, and /dev inside the guest so that
// commands run via guestfish "command" (which chroots into the guest)
// can access virtual filesystems. This matches what virt-v2v does
// before running dracut and other system tools.
func (b *Backend) mountVirtualFS() error {
	script := strings.Join([]string{
		"-mkdir-p /proc",
		"-mkdir-p /sys",
		"-mkdir-p /dev",
		`-mount-vfs "" proc /proc /proc`,
		`-mount-vfs "" sysfs /sys /sys`,
		`-mount-vfs "" devtmpfs /dev /dev`,
	}, "\n") + "\n"
	_, err := b.session.remoteScript(script)
	return err
}

func (b *Backend) unmountVirtualFS() {
	script := "-umount /dev\n-umount /sys\n-umount /proc\n"
	_, _ = b.session.remoteScript(script)
}

// RunCommand executes cmd inside the guest via guestfish "sh".
// We use "sh" instead of "command" because it allows a 2>&1 redirect
// that merges stderr into stdout. Both "command" and "sh" only return
// the command's stdout; without the redirect, stderr (where dracut
// and other tools report errors) is silently discarded by libguestfs,
// making failures invisible to the caller.
// Virtual filesystems (/proc, /sys, /dev) are mounted before the
// command runs and unmounted afterwards, matching virt-v2v behaviour.
func (b *Backend) RunCommand(_ string, cmd []string) ([]byte, error) {
	if err := b.ensureMounted(); err != nil {
		return nil, err
	}
	if err := b.mountVirtualFS(); err != nil {
		slog.Warn("failed to mount virtual filesystems for guest command", "error", err)
	}
	defer b.unmountVirtualFS()

	var shell strings.Builder
	for i, arg := range cmd {
		if i > 0 {
			shell.WriteByte(' ')
		}
		shell.WriteString(shellQuote(arg))
	}
	shell.WriteString(" 2>&1")

	var script strings.Builder
	script.WriteString("sh ")
	script.WriteString(quoteGuestfish(shell.String()))
	script.WriteByte('\n')
	s := script.String()
	return b.withRecovery(func() ([]byte, error) {
		return b.session.remoteScript(s)
	})
}

func (b *Backend) DeviceRead(device string, offset int64, size int) ([]byte, error) {
	if err := b.ensureSession(); err != nil {
		return nil, err
	}
	out, err := b.session.remote("pread-device", device, strconv.Itoa(size), strconv.FormatInt(offset, 10))
	if err != nil {
		return nil, err
	}
	data := out
	if cleaned, herr := hex.DecodeString(strings.TrimSpace(string(out))); herr == nil && len(cleaned) == size {
		data = cleaned
	}
	if len(data) > size {
		data = data[:size]
	}
	return data, nil
}

func (b *Backend) DeviceWrite(device string, offset int64, data []byte) error {
	if err := b.ensureSession(); err != nil {
		return err
	}
	tmp, err := os.CreateTemp("", "kc-guest-pdev-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()

	var script strings.Builder
	script.WriteString("upload ")
	script.WriteString(quoteGuestfish(tmpPath))
	script.WriteByte(' ')
	script.WriteString(quoteGuestfish("/tmp/.kc-pdev"))
	script.WriteByte('\n')
	script.WriteString("pwrite-device ")
	script.WriteString(quoteGuestfish(device))
	script.WriteByte(' ')
	script.WriteString(quoteGuestfish("/tmp/.kc-pdev"))
	script.WriteByte(' ')
	script.WriteString(strconv.FormatInt(offset, 10))
	script.WriteByte('\n')
	_, err = b.session.remoteScript(script.String())
	return err
}

func (b *Backend) Sync() error {
	return nil
}

func (b *Backend) Release() error {
	return b.teardown()
}

func (b *Backend) Teardown() error {
	return b.teardown()
}

// TeardownDiscard is identical to Teardown for the guestfs backend: the
// appliance owns the disk images, so closing the session discards uncommitted
// state automatically.
func (b *Backend) TeardownDiscard() error {
	return b.Teardown()
}

func (b *Backend) UnmountFilesystems() error {
	var firstErr error
	if b.probeActive != "" {
		if err := b.ProbeUnmount(b.probeActive); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if b.session != nil {
		if _, err := b.session.remoteScript("-umount-all\n"); err != nil && firstErr == nil {
			slog.Debug("guestfs umount-all", "error", err)
		}
	}
	b.mounted = false
	b.mountsActive = false
	b.mountSpecs = nil
	return firstErr
}

func (b *Backend) ReleaseDevices() error {
	var firstErr error
	if b.session != nil {
		for i := len(b.cryptMaps) - 1; i >= 0; i-- {
			if err := b.CloseCrypt(b.cryptMaps[i]); err != nil {
				slog.Debug("guestfs cryptsetup-close", "mapper", b.cryptMaps[i], "error", err)
				if firstErr == nil {
					firstErr = err
				}
			}
		}
		b.cryptMaps = nil
		if err := b.session.close(); err != nil && firstErr == nil {
			firstErr = err
		}
		b.session = nil
	}
	return firstErr
}

func (b *Backend) teardown() error {
	var firstErr error
	if err := b.UnmountFilesystems(); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := b.ReleaseDevices(); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// TeardownMountRoot best-effort cleans orphaned guestfs resources.
func TeardownMountRoot(mountRoot string) error {
	return clearDir(mountRoot)
}
