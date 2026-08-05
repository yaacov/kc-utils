//go:build linux

package mount

import (
	"fmt"
	"os/exec"
	"strings"
)

// Manager tracks mounted filesystems and unmounts them in reverse order.
type Manager struct {
	entries []Entry
}

// Entry records a single mount.
type Entry struct {
	Device     string
	Mountpoint string
	FSType     string
}

func NewManager() *Manager {
	return &Manager{}
}

// Mount mounts a device at the given mountpoint via mount(8).
func (m *Manager) Mount(device, mountpoint, fstype string, readOnly bool) error {
	opts := mountOptions(fstype, readOnly)
	args := []string{"-t", fstype, "-o", opts, device, mountpoint}
	if out, err := exec.Command("mount", args...).CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("mount %s on %s (type %s): %w: %s", device, mountpoint, fstype, err, msg)
		}
		return fmt.Errorf("mount %s on %s (type %s): %w", device, mountpoint, fstype, err)
	}

	m.entries = append(m.entries, Entry{
		Device:     device,
		Mountpoint: mountpoint,
		FSType:     fstype,
	})
	return nil
}

// Remount changes mount flags on an already-mounted filesystem via mount(8).
func (m *Manager) Remount(mountpoint string, readWrite bool) error {
	mode := "ro"
	if readWrite {
		mode = "rw"
	}
	args := []string{"-o", "remount," + mode, mountpoint}
	if out, err := exec.Command("mount", args...).CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("remount %s (%s): %w: %s", mountpoint, mode, err, msg)
		}
		return fmt.Errorf("remount %s (%s): %w", mountpoint, mode, err)
	}
	return nil
}

// Unmount unmounts a single mountpoint via umount(8).
func Unmount(mountpoint string) error {
	if out, err := exec.Command("umount", mountpoint).CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("unmount %s: %w: %s", mountpoint, err, msg)
		}
		return fmt.Errorf("unmount %s: %w", mountpoint, err)
	}
	return nil
}

// UnmountAll unmounts all filesystems in reverse order (deepest first).
func (m *Manager) UnmountAll() error {
	var firstErr error
	for i := len(m.entries) - 1; i >= 0; i-- {
		if err := Unmount(m.entries[i].Mountpoint); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	m.entries = nil
	return firstErr
}

// Entries returns the current mount list.
func (m *Manager) Entries() []Entry {
	return m.entries
}

func mountOptions(fstype string, readOnly bool) string {
	parts := []string{"nodev", "nosuid", "noexec"}
	if readOnly {
		parts = append(parts, "ro")
	}
	if fs := fsOptions(fstype); fs != "" {
		parts = append(parts, fs)
	}
	return strings.Join(parts, ",")
}

func fsOptions(fstype string) string {
	switch fstype {
	case "ext4", "ext3", "ext2":
		return "norecovery"
	case "xfs":
		return "norecovery"
	case "ntfs3":
		return "force"
	default:
		return ""
	}
}
