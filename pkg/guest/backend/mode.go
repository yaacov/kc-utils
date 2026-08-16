//go:build unix

package backend

import (
	"fmt"
	"strings"
)

// Mode selects how privileged guest disk operations are performed.
// Values are registry keys for Factories ("direct", "guestfs", "qemu", …).
type Mode string

const (
	// ModeDirect uses host syscalls (mount, losetup, LVM, cryptsetup).
	// Requires CAP_SYS_ADMIN / privileged pods. Registered on Linux only.
	ModeDirect Mode = "direct"
	// ModeGuestfs uses a libguestfs guestfish --listen appliance.
	// Guest filesystems stay inside the appliance; I/O uses guestfish RPC.
	// Suitable for unprivileged pods with /dev/kvm. Registered on Linux only.
	ModeGuestfs Mode = "guestfs"
	// ModeQemu boots a shipped kernel+initramfs under QEMU and talks to
	// kc-agent over a virtio-serial Unix socket. Registered on Linux and Darwin.
	ModeQemu Mode = "qemu"
)

func (m Mode) String() string {
	return string(m)
}

// ParseMode validates name against registered Factories.
// Empty name is an error; there is no default backend.
func ParseMode(name string) (Mode, error) {
	resolved, err := RequireBackend(name)
	if err != nil {
		return "", err
	}
	return Mode(resolved), nil
}

// RequireBackend returns a trimmed backend name after checking it is registered.
func RequireBackend(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		avail := AvailableBackends()
		if len(avail) == 0 {
			return "", fmt.Errorf("--backend is required (no backends registered)")
		}
		return "", fmt.Errorf("--backend is required (available: %s)", strings.Join(avail, ", "))
	}
	if _, err := LookupFactory(name); err != nil {
		return "", err
	}
	return name, nil
}
