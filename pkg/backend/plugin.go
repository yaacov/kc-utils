//go:build unix

package backend

import (
	"github.com/yaacov/kc-utils/pkg/common/plugin"
	"github.com/yaacov/kc-utils/pkg/common/types"
)

// Requirements declares runtime prerequisites for a backend plugin.
type Requirements struct {
	Linux     bool
	Root      bool
	KVM       bool
	Guestfish bool
	// QEMU requires a qemu-system-<hostarch> (or qemu-kvm) binary in PATH.
	QEMU bool
	// Accel requires hardware virtualization (KVM on Linux, HVF on macOS).
	// The qemu backend does not set this: it falls back to TCG emulation when
	// no accelerator is present, so acceleration is preferred but not required.
	Accel bool
}

// Plugin is a registered backend implementation factory.
type Plugin interface {
	Name() string
	Requirements() Requirements
	Available() bool
	New() Backend
	NewMounted(disks []types.DiskSpec, mountRoot string, infos []types.DiskInfo) (Backend, error)
	TeardownMountRoot(mountRoot string) error
}

// SharedListener is a cross-stage guestfish --listen session handle.
type SharedListener interface {
	Env() []string
	Alive() bool
	Close() error
}

// SharedListenerPlugin is implemented by backends that keep a VM-resident
// session alive across pipeline stages (guestfs, qemu). disks are attached at
// boot; guestfs ignores them (it adds drives lazily), qemu boots with them.
type SharedListenerPlugin interface {
	Plugin
	StartSharedListener(disks []types.DiskSpec) (SharedListener, error)
}

// Plugins is the global backend plugin registry.
var Plugins = plugin.NewRegistry[string, Plugin]()
