//go:build unix

package qemu

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/backend"
)

// hostAccelAvailable reports whether hardware virtualization (KVM/HVF) is
// available on the host, so the session can prefer it at launch and fall back
// to TCG otherwise.
func hostAccelAvailable() bool {
	return backend.Probes.HasAccel()
}

// applianceProbeRoot is where a candidate device is temporarily read-only
// mounted for OS inspection (one probe at a time).
const applianceProbeRoot = "/mnt/probe"

// probeRemotePaths are the OS-identity markers copied out to the host for
// inspection; mirrors the guestfs backend allowlist.
var probeRemotePaths = []string{
	"/etc/os-release",
	"/usr/lib/os-release",
	"/etc/redhat-release",
	"/etc/debian_version",
	"/Windows/System32/config/SYSTEM",
	"/Windows/System32/config/SOFTWARE",
	"/windows/system32/config/SYSTEM",
	"/windows/system32/config/SOFTWARE",
	"/WINDOWS/System32/config/SYSTEM",
	"/WINDOWS/System32/config/SOFTWARE",
}

// ProbeMount read-only mounts device in the appliance, copies OS-identity
// markers to the host directory hostMountPoint for inspection, then unmounts.
func (b *Backend) ProbeMount(device, fstype, hostMountPoint string) error {
	if b.probeActive != "" {
		_ = b.ProbeUnmount(b.probeActive)
	}
	if err := os.MkdirAll(hostMountPoint, 0o755); err != nil {
		return err
	}
	if err := clearDir(hostMountPoint); err != nil {
		return err
	}

	if _, err := b.session.client.run("mkdir", "-p", applianceProbeRoot); err != nil {
		return fmt.Errorf("mkdir probe root: %w", err)
	}
	argv := []string{"mount", "-r"}
	if fstype != "" {
		argv = append(argv, "-t", fstype)
	}
	argv = append(argv, device, applianceProbeRoot)
	if _, err := b.session.client.run(argv...); err != nil {
		return fmt.Errorf("probe mount %s: %w", device, err)
	}
	defer func() {
		if _, err := b.session.client.run("umount", applianceProbeRoot); err != nil {
			// Non-fatal: the next probe re-mounts after unmounting.
			_ = err
		}
	}()

	for _, marker := range probeRemotePaths {
		if err := b.copyMarkerOut(marker, hostMountPoint); err != nil {
			return err
		}
	}

	pruneEmptyWindowsProbeDirs(hostMountPoint)
	b.probeActive = hostMountPoint
	return nil
}

// copyMarkerOut copies one marker file (relative to the probe root) to the host
// inspection directory, preserving its relative path. Missing markers are
// silently skipped.
func (b *Backend) copyMarkerOut(marker, hostMountPoint string) error {
	src := path.Join(applianceProbeRoot, marker)
	st, err := b.session.client.stat(src)
	if err != nil || !st.Exists || st.IsDir {
		return nil
	}
	data, err := b.session.client.readFile(src)
	if err != nil {
		return nil
	}
	dest := filepath.Join(hostMountPoint, filepath.FromSlash(strings.TrimPrefix(marker, "/")))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0o644)
}

// ProbeUnmount removes the host inspection directory populated by ProbeMount.
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

// pruneEmptyWindowsProbeDirs removes marker directories left empty when a device
// is not the OS it might have been (mirrors the guestfs backend).
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

// clearDir removes the contents of dir without removing dir itself.
func clearDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
