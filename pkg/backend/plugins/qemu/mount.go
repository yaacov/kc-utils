//go:build unix

package qemu

import (
	"fmt"
	"path/filepath"
	"strings"
)

// hostMountFromGuest builds the host mount point that corresponds to a guest
// mount path, used when re-establishing a recorded mount plan. The guest path
// is cleaned under "/" first so ".." cannot walk above mountRoot.
func hostMountFromGuest(mountRoot, guestMount string) string {
	if guestMount == "/" || guestMount == "" {
		return mountRoot
	}
	confined := filepath.Clean("/" + filepath.FromSlash(guestMount))
	rel := strings.TrimPrefix(confined, "/")
	if rel == "" {
		return mountRoot
	}
	return filepath.Join(mountRoot, rel)
}

// pathUnderRoot reports whether p is root or a descendant of root after Clean.
func pathUnderRoot(root, p string) bool {
	root = filepath.Clean(root)
	p = filepath.Clean(p)
	if root == "" || root == "." {
		return false
	}
	if p == root {
		return true
	}
	return strings.HasPrefix(p, root+"/")
}

// Mount eagerly mounts device at the appliance path corresponding to
// hostMountPoint. The mount happens inside the appliance via the agent; only
// the appliance-side bookkeeping is kept host-side for reverse teardown.
func (b *Backend) Mount(device, hostMountPoint, fstype string, readOnly bool) error {
	target := applianceMountPath(b.mountRoot, hostMountPoint)

	for _, m := range b.mounts {
		if m.AppliancePath == target {
			if m.Device == device {
				return nil // already mounted here; idempotent
			}
			return fmt.Errorf("mount conflict: %s already mounted at %s, cannot mount %s", m.Device, target, device)
		}
	}

	if _, err := b.session.client.run("mkdir", "-p", target); err != nil {
		return fmt.Errorf("mkdir %s: %w", target, err)
	}

	argv := []string{"mount"}
	if readOnly {
		argv = append(argv, "-r")
	}
	if fstype != "" {
		argv = append(argv, "-t", fstype)
	}
	argv = append(argv, device, target)
	if _, err := b.session.client.run(argv...); err != nil {
		return fmt.Errorf("mount %s at %s: %w", device, target, err)
	}

	b.mounts = append(b.mounts, mountEntry{Device: device, AppliancePath: target})
	return nil
}

// UnmountAll unmounts all recorded mounts in reverse order. Entries that fail
// to unmount are retained (in their original order) so a later retry or Teardown
// can process them again, rather than being silently forgotten.
func (b *Backend) UnmountAll() error {
	var firstErr error
	failed := make([]bool, len(b.mounts))
	for i := len(b.mounts) - 1; i >= 0; i-- {
		if _, err := b.session.client.run("umount", b.mounts[i].AppliancePath); err != nil {
			failed[i] = true
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	var remaining []mountEntry
	for i, m := range b.mounts {
		if failed[i] {
			remaining = append(remaining, m)
		}
	}
	b.mounts = remaining
	return firstErr
}

// FSTrim discards unused blocks on the filesystem mounted at mountpoint.
func (b *Backend) FSTrim(mountpoint string) error {
	target := applianceMountPath(b.mountRoot, mountpoint)
	_, err := b.session.client.run("fstrim", "-v", target)
	return err
}
