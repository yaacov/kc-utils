//go:build unix

package initramfs

import (
	"bytes"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest"
)

// InjectVirtioModules rebuilds the initramfs for the selected kernel to ensure
// virtio drivers are included for early boot. The existing initramfs is backed
// up before the rebuild. --add-drivers is a single dracut-install -m list: one
// missing module fails the whole set, so only modules needed to boot on virtio
// are passed (not optional display drivers such as bochs).
func InjectVirtioModules(guestRoot string, kernel *types.KernelInfo) error {
	if kernel == nil {
		return fmt.Errorf("no kernel selected")
	}
	if kernel.Path == "" {
		return fmt.Errorf("kernel %s has no vmlinuz, cannot rebuild initramfs", kernel.Version)
	}

	initrdPath := kernel.InitrdPath
	if initrdPath == "" {
		initrdPath = inferInitrdPath(guestRoot, kernel.Version)
		slog.Info("inferred initrd path", "path", initrdPath, "kernel", kernel.Version)
	}

	initrdHostPath := filepath.Join(guestRoot, initrdPath)
	if guest.FileExists(initrdHostPath) {
		_ = guest.FileCopy(initrdHostPath, initrdHostPath+".pre-v2v")
	}

	virtioDrivers := strings.Join([]string{
		"virtio", "virtio_ring", "virtio_blk", "virtio_scsi", "virtio_net", "virtio_pci",
		"xts",
	}, " ")

	slog.Info("running dracut", "kernel", kernel.Version)
	out, err := guest.RunInGuest(guestRoot, dracutArgv(initrdPath, kernel.Version, virtioDrivers))
	if err == nil && dracutReportedFailure(out) {
		err = fmt.Errorf("dracut reported FAILED")
	}
	if err == nil {
		slog.Info("dracut completed", "kernel", kernel.Version, "output", string(out))
		if verifyErr := verifyInitramfsRebuilt(initrdHostPath); verifyErr != nil {
			return verifyErr
		}
		slog.Info("rebuilt initramfs with dracut", "kernel", kernel.Version)
		return nil
	}
	slog.Warn("dracut failed", "kernel", kernel.Version, "error", err, "output", string(out))

	initramfsToolsDir := filepath.Join(guestRoot, "etc", "initramfs-tools")
	if !guest.FileExists(initramfsToolsDir) {
		return fmt.Errorf("dracut failed and no initramfs-tools found for kernel %s: %w", kernel.Version, err)
	}

	slog.Info("trying Debian initramfs tools", "kernel", kernel.Version)
	if err := ensureInitramfsToolsModules(guestRoot, virtioDrivers); err != nil {
		return err
	}

	out, err = guest.RunInGuest(guestRoot, []string{
		"update-initramfs", "-u", "-k", kernel.Version,
	})
	if err == nil {
		slog.Info("rebuilt initramfs with update-initramfs", "kernel", kernel.Version)
		return nil
	}
	slog.Warn("update-initramfs failed, trying mkinitramfs", "kernel", kernel.Version, "error", err, "output", string(out))

	_, err = guest.RunInGuest(guestRoot, []string{
		"mkinitramfs", "-o", initrdPath, kernel.Version,
	})
	if err == nil {
		slog.Info("rebuilt initramfs with mkinitramfs", "kernel", kernel.Version)
		return nil
	}

	slog.Error("all initramfs rebuild methods failed, VM may not boot without virtio drivers in initramfs", "kernel", kernel.Version, "error", err)
	return fmt.Errorf("all initramfs rebuild methods failed for kernel %s: %w", kernel.Version, err)
}

// verifyInitramfsRebuilt checks that the initramfs was actually modified by
// comparing it to the pre-rebuild backup. Dracut can exit 0 without doing
// anything useful (e.g., when running inside the guestfs appliance chroot
// without a proper environment), so we verify the result.
func verifyInitramfsRebuilt(initrdHostPath string) error {
	backupPath := initrdHostPath + ".pre-v2v"
	origData, err := guest.FileRead(backupPath)
	if err != nil {
		slog.Warn("could not read initramfs backup for verification", "path", backupPath, "error", err)
		return nil
	}
	newData, err := guest.FileRead(initrdHostPath)
	if err != nil {
		return fmt.Errorf("initramfs missing after dracut: %w", err)
	}
	if len(newData) == len(origData) {
		slog.Warn("initramfs unchanged after dracut",
			"path", initrdHostPath, "size", len(newData),
		)
		return fmt.Errorf("dracut exited 0 but initramfs is unchanged (%d bytes); rebuild likely failed silently", len(newData))
	}
	slog.Info("initramfs size changed after rebuild",
		"before", len(origData), "after", len(newData),
	)
	return nil
}

// inferInitrdPath determines the most likely initrd path for a kernel version
// by checking common naming conventions.
func inferInitrdPath(guestRoot, version string) string {
	candidates := []string{
		"/boot/initramfs-" + version + ".img",
		"/boot/initrd.img-" + version,
		"/boot/initrd-" + version,
	}
	for _, c := range candidates {
		if guest.FileExists(filepath.Join(guestRoot, c)) {
			return c
		}
	}
	// Default to RPM convention when no existing file is found.
	return "/boot/initramfs-" + version + ".img"
}

func dracutReportedFailure(out []byte) bool {
	return bytes.Contains(out, []byte("dracut: FAILED"))
}

// dracutArgv rebuilds a generic virtio-capable initramfs for kernelVersion.
// --no-hostonly keeps dracut from following whatever /sys is visible (appliance
// or source hypervisor hardware).
func dracutArgv(initrdPath, kernelVersion, virtioDrivers string) []string {
	return []string{
		"dracut", "--force",
		"--no-hostonly", "--no-hostonly-cmdline",
		"--add-drivers", virtioDrivers,
		initrdPath, kernelVersion,
	}
}

// ensureInitramfsToolsModules adds virtio module names to
// /etc/initramfs-tools/modules so update-initramfs includes them.
func ensureInitramfsToolsModules(guestRoot, virtioDrivers string) error {
	modulesFile := filepath.Join(guestRoot, "etc", "initramfs-tools", "modules")
	if !guest.FileExists(filepath.Dir(modulesFile)) {
		return nil
	}
	var content string
	if existing, err := guest.FileRead(modulesFile); err == nil {
		content = string(existing)
	}
	changed := false
	for _, m := range strings.Fields(virtioDrivers) {
		if !strings.Contains(content, m) {
			content = strings.TrimRight(content, "\n") + "\n" + m + "\n"
			changed = true
		}
	}
	if changed {
		if err := guest.FileWrite(modulesFile, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write initramfs-tools modules %s: %w", modulesFile, err)
		}
	}
	return nil
}
