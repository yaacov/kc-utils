//go:build linux

package guest

import (
	"strings"
)

// Mode selects how privileged guest disk operations are performed.
// Values are registry keys for Factories ("direct", "guestfs", …).
type Mode string

const (
	// ModeDirect uses host syscalls (mount, losetup, LVM, cryptsetup).
	// Requires CAP_SYS_ADMIN / privileged pods.
	ModeDirect Mode = "direct"
	// ModeGuestfs uses a libguestfs guestfish --listen appliance.
	// Guest filesystems stay inside the appliance; I/O uses guestfish RPC.
	// Suitable for unprivileged pods with /dev/kvm.
	ModeGuestfs Mode = "guestfs"
)

func (m Mode) String() string {
	if m == "" {
		return string(ModeDirect)
	}
	return string(m)
}

// ParseMode validates name against registered Factories.
// Empty name defaults to ModeDirect.
func ParseMode(name string) (Mode, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = string(ModeDirect)
	}
	if _, err := LookupFactory(name); err != nil {
		return "", err
	}
	return Mode(name), nil
}
