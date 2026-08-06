//go:build linux

package guestfs

import (
	"os"
	"path/filepath"
	"strings"
)

func (b *Backend) ProbeMount(device, _ string, hostMountPoint string) error {
	if b.probeActive != "" {
		_ = b.ProbeUnmount(b.probeActive)
	}
	if err := os.MkdirAll(hostMountPoint, 0o755); err != nil {
		return err
	}
	if err := clearDir(hostMountPoint); err != nil {
		return err
	}

	for _, remote := range probeRemotePaths {
		localDir := filepath.Join(hostMountPoint, filepath.Dir(strings.TrimPrefix(remote, "/")))
		if err := os.MkdirAll(localDir, 0o755); err != nil {
			return err
		}
	}

	if err := b.ensureSession(); err != nil {
		return err
	}

	var script strings.Builder
	script.WriteString("-umount-all\n")
	script.WriteString("-mount ")
	script.WriteString(quoteGuestfish(device))
	script.WriteString(" /\n")
	for _, remote := range probeRemotePaths {
		localDir := filepath.Join(hostMountPoint, filepath.Dir(strings.TrimPrefix(remote, "/")))
		script.WriteString("-copy-out ")
		script.WriteString(quoteGuestfish(remote))
		script.WriteByte(' ')
		script.WriteString(quoteGuestfish(localDir))
		script.WriteByte('\n')
	}
	script.WriteString("-umount-all\n")

	if _, err := b.session.remoteScriptSoft(script.String()); err != nil {
		return err
	}

	pruneEmptyWindowsProbeDirs(hostMountPoint)

	b.probeActive = hostMountPoint
	return nil
}

var probeRemotePaths = []string{
	"/etc/os-release",
	"/etc/redhat-release",
	"/etc/debian_version",
	"/Windows/System32/config/SYSTEM",
	"/Windows/System32/config/SOFTWARE",
	"/windows/system32/config/SYSTEM",
	"/windows/system32/config/SOFTWARE",
	"/WINDOWS/System32/config/SYSTEM",
	"/WINDOWS/System32/config/SOFTWARE",
}

func pruneEmptyWindowsProbeDirs(hostMountPoint string) {
	for _, top := range []string{"Windows", "windows", "WINDOWS"} {
		hive := filepath.Join(hostMountPoint, top, "System32", "config", "SYSTEM")
		if _, err := os.Stat(hive); err != nil {
			hive2 := filepath.Join(hostMountPoint, top, "system32", "config", "SYSTEM")
			if _, err2 := os.Stat(hive2); err2 != nil {
				_ = os.RemoveAll(filepath.Join(hostMountPoint, top))
			}
		}
	}
	etc := filepath.Join(hostMountPoint, "etc")
	if entries, err := os.ReadDir(etc); err == nil && len(entries) == 0 {
		_ = os.Remove(etc)
	}
}

func (b *Backend) ProbeUnmount(hostMountPoint string) error {
	if hostMountPoint == "" {
		return nil
	}
	err := os.RemoveAll(hostMountPoint)
	if b.probeActive == hostMountPoint {
		b.probeActive = ""
	}
	return err
}
