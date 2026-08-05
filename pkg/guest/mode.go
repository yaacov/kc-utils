//go:build linux

package guest

// Mode selects how privileged guest disk operations are performed.
type Mode int

const (
	// ModeDirect uses host syscalls (mount, losetup, LVM, cryptsetup).
	// Requires CAP_SYS_ADMIN / privileged pods.
	ModeDirect Mode = iota
	// ModeGuestfs uses a libguestfs guestfish --listen appliance.
	// Guest filesystems stay inside the appliance; I/O uses guestfish RPC.
	// Suitable for unprivileged pods with /dev/kvm.
	ModeGuestfs
)

func (m Mode) String() string {
	switch m {
	case ModeGuestfs:
		return "guestfs"
	default:
		return "direct"
	}
}

// ModeFromBool maps the historical --guestfs flag to Mode.
func ModeFromBool(useGuestfs bool) Mode {
	if useGuestfs {
		return ModeGuestfs
	}
	return ModeDirect
}
