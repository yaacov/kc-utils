//go:build unix

package core

import (
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/yaacov/kc-utils/pkg/guest/runtime"
)

// ---- Mounting ----------------------------------------------------------

// mountOptions returns mount(8) -o options for fstype. Read-only inspection
// avoids journal recovery/replay; ntfs3 needs "force" to mount dirty volumes.
func mountOptions(fstype string, readOnly bool) string {
	parts := []string{"nodev", "nosuid", "noexec"}
	if readOnly {
		parts = append(parts, "ro")
	}
	switch fstype {
	case "ext4", "ext3", "ext2", "xfs":
		parts = append(parts, "norecovery")
	case "ntfs3":
		parts = append(parts, "force")
	}
	return strings.Join(parts, ",")
}

func (b *Backend) mountAt(device, mountpoint, fstype string, readOnly bool) error {
	if err := b.rt.MkdirAll(mountpoint, 0o755); err != nil {
		return err
	}
	opts := mountOptions(fstype, readOnly)
	args := []string{"mount"}
	if fstype != "" && fstype != "auto" {
		args = append(args, "-t", fstype)
	}
	args = append(args, "-o", opts, device, mountpoint)
	if _, err := b.run(args...); err != nil {
		return err
	}
	return nil
}

func (b *Backend) unmountAt(mountpoint string) error {
	_, err := b.run("umount", mountpoint)
	return err
}

func (b *Backend) Mount(device, hostMountPoint, fstype string, readOnly bool) error {
	if err := b.mountAt(device, hostMountPoint, fstype, readOnly); err != nil {
		return err
	}
	b.mounts = append(b.mounts, hostMountPoint)
	return nil
}

func (b *Backend) UnmountAll() error {
	var first error
	for _, m := range slices.Backward(b.mounts) {
		if err := b.unmountAt(m); err != nil && first == nil {
			first = err
		}
	}
	b.mounts = nil
	return first
}

func (b *Backend) ProbeMount(device, fstype, hostMountPoint string) error {
	if err := b.mountAt(device, hostMountPoint, fstype, true); err != nil {
		return err
	}
	b.probes = append(b.probes, hostMountPoint)
	return nil
}

func (b *Backend) ProbeUnmount(hostMountPoint string) error {
	err := b.unmountAt(hostMountPoint)
	filtered := b.probes[:0]
	for _, m := range b.probes {
		if m != hostMountPoint {
			filtered = append(filtered, m)
		}
	}
	b.probes = filtered
	return err
}

// UnmountProbes unmounts probe mounts only (used by Release before convert).
func (b *Backend) UnmountProbes() error {
	var first error
	for _, m := range slices.Backward(b.probes) {
		if err := b.unmountAt(m); err != nil && first == nil {
			first = err
		}
	}
	b.probes = nil
	return first
}

// UnmountFilesystems unmounts probe mounts and every filesystem mounted under
// the guest root, keeping LUKS/LVM open. It works from /proc/mounts rather than
// tracked state, so it is correct even when this backend re-attached to a
// session whose mounts were created by an earlier process.
func (b *Backend) UnmountFilesystems() error {
	first := b.UnmountProbes()
	if err := b.UnmountUnder(b.guestRoot); err != nil && first == nil {
		first = err
	}
	b.mounts = nil
	return first
}

// UnmountUnder unmounts every filesystem mounted at or beneath mountRoot,
// deepest first, reading the mount list from the runtime's /proc/mounts.
func (b *Backend) UnmountUnder(mountRoot string) error {
	if mountRoot == "" {
		return nil
	}
	data, err := b.rt.ReadFile("/proc/mounts")
	if err != nil {
		return err
	}
	var mps []string
	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		mp := fields[1]
		if mp == mountRoot || strings.HasPrefix(mp, mountRoot+"/") {
			mps = append(mps, mp)
		}
	}
	sort.Slice(mps, func(i, j int) bool {
		return strings.Count(mps[i], "/") > strings.Count(mps[j], "/")
	})
	var first error
	for _, mp := range mps {
		if err := b.unmountAt(mp); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// ---- Filesystem checks -------------------------------------------------

func (b *Backend) FSCheck(device, fstype string) error {
	switch fstype {
	case "ext4", "ext3", "ext2":
		// e2fsck exit codes are a bitmask: 0 = clean, 1 = errors corrected,
		// 2 = errors corrected + reboot recommended. All three are successful
		// checks. Only bit 2+ (>= 4: errors left uncorrected, or an
		// operational/usage failure) is fatal. Treating 1/2 as failure would
		// abort the pipeline on a dirty filesystem that fsck successfully
		// repaired (the guestfs backend also tolerates 1/2).
		res, err := b.runStatus("e2fsck", "-f", "-y", device)
		if err != nil {
			return err
		}
		if res.Exit >= 4 {
			return fmt.Errorf("e2fsck: exit %d: %s", res.Exit, strings.TrimSpace(string(res.Combined())))
		}
		return nil
	case "xfs":
		_, err := b.run("xfs_repair", device)
		return err
	case "btrfs":
		_, err := b.run("btrfs", "check", device)
		return err
	case "ntfs3", "ntfs":
		_, err := b.run("ntfsfix", "-d", device)
		return err
	default:
		return nil
	}
}

func (b *Backend) FSTrim(mountpoint string) error {
	_, err := b.run("fstrim", "-v", mountpoint)
	return err
}

// ---- LUKS / Clevis -----------------------------------------------------

func (b *Backend) Decrypt(device, keyFile, mapperName string) (string, error) {
	argv := []string{"cryptsetup", "open", "--type", "luks", device, mapperName}
	if keyFile != "" {
		// keyFile is a host artifact; push its bytes into the runtime so the
		// same code works whether the runtime is local or in the appliance.
		key, err := readHostFile(keyFile)
		if err != nil {
			return "", err
		}
		keyPath := "/tmp/kc-luks-key-" + sanitize(mapperName)
		if err := b.rt.WriteFile(keyPath, key, 0o600); err != nil {
			return "", err
		}
		defer func() { _ = b.rt.Remove(keyPath) }()
		argv = append(argv, "--key-file", keyPath)
	}
	if _, err := b.run(argv...); err != nil {
		return "", err
	}
	b.cryptMaps = append(b.cryptMaps, mapperName)
	return "/dev/mapper/" + mapperName, nil
}

func (b *Backend) UnlockClevis(device, mapperName string) (string, error) {
	// Pre-flight for a clearer error when no Clevis binding exists.
	if _, err := b.run("clevis", "luks", "list", "-d", device); err != nil {
		return "", fmt.Errorf("clevis pre-flight on %s: %w", device, err)
	}
	if _, err := b.run("clevis", "luks", "unlock", "-d", device, "-n", mapperName); err != nil {
		return "", fmt.Errorf("clevis unlock %s: %w", device, err)
	}
	b.cryptMaps = append(b.cryptMaps, mapperName)
	return "/dev/mapper/" + mapperName, nil
}

func (b *Backend) CloseCrypt(mapperName string) error {
	_, err := b.run("cryptsetup", "close", mapperName)
	return err
}

// ---- Command execution in the guest ------------------------------------

func (b *Backend) RunCommand(guestRoot string, cmd []string) ([]byte, error) {
	root := b.guestRoot
	if guestRoot != "" {
		root = b.host(guestRoot)
	}
	argv := append([]string{"chroot", root}, cmd...)
	res, err := b.rt.Run(&runtime.CommandSpec{Argv: argv})
	if err != nil {
		return res.Combined(), fmt.Errorf("chroot %v: %w", cmd, err)
	}
	if res.Exit == 0 {
		return res.Combined(), nil
	}
	// Unprivileged hosts may deny chroot(2); retry inside a user namespace.
	if chrootNotPermitted(res.Combined()) {
		unshareArgv := append([]string{"unshare", "-r", "chroot", root}, cmd...)
		res2, err2 := b.rt.Run(&runtime.CommandSpec{Argv: unshareArgv})
		if err2 != nil {
			return res2.Combined(), fmt.Errorf("unshare chroot %v: %w", cmd, err2)
		}
		if res2.Exit != 0 {
			return res2.Combined(), fmt.Errorf("chroot %v: exit %d", cmd, res2.Exit)
		}
		return res2.Combined(), nil
	}
	return res.Combined(), fmt.Errorf("chroot %v: exit %d", cmd, res.Exit)
}

func chrootNotPermitted(out []byte) bool {
	return strings.Contains(string(out), "Operation not permitted")
}

// ---- Windows hive merge ------------------------------------------------

func (b *Backend) MergeHive(guestPath string, reg []byte) error {
	if len(reg) == 0 {
		return nil
	}
	tmp := "/tmp/kc-reg-" + sanitize(guestPath) + ".reg"
	if err := b.rt.WriteFile(tmp, reg, 0o600); err != nil {
		return fmt.Errorf("merge hive temp: %w", err)
	}
	defer func() { _ = b.rt.Remove(tmp) }()
	hive := b.host(guestPath)
	if _, err := b.run("hivexregedit", "--merge", hive, tmp); err != nil {
		return err
	}
	return nil
}

// ---- Device lifecycle helpers (for backend Teardown/ReleaseDevices) -----

// CloseCryptMaps closes all tracked LUKS mappers in reverse order.
func (b *Backend) CloseCryptMaps() error {
	var first error
	for _, m := range slices.Backward(b.cryptMaps) {
		if _, err := b.run("cryptsetup", "close", m); err != nil && first == nil {
			first = err
		}
	}
	b.cryptMaps = nil
	return first
}

// CloseAllCryptMaps closes every /dev/mapper entry via cryptsetup (best-effort;
// LVM/non-LUKS mappings simply fail and are ignored). Unlike CloseCryptMaps it
// needs no tracked state, so it also cleans mappers opened by an earlier process.
func (b *Backend) CloseAllCryptMaps() {
	entries, err := b.rt.ReadDir("/dev/mapper")
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.Name == "control" {
			continue
		}
		_, _ = b.rt.Run(&runtime.CommandSpec{Argv: []string{"cryptsetup", "close", e.Name}})
	}
	b.cryptMaps = nil
}

// DeactivateLVM deactivates all volume groups.
func (b *Backend) DeactivateLVM() error {
	_, err := b.run("vgchange", "-an")
	return err
}

func sanitize(s string) string {
	out := make([]byte, 0, len(s))
	for _, c := range s {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			out = append(out, byte(c))
		default:
			out = append(out, '_')
		}
	}
	return filepath.Clean(string(out))
}
