//go:build unix

package guest

import (
	"fmt"

	"github.com/yaacov/kc-utils/pkg/backend"
	"github.com/yaacov/kc-utils/pkg/common/types"
)

const (
	EnvGuestfishPID   = "GUESTFISH_PID"
	EnvKCGuestfishPID = "KC_GUESTFISH_PID"
	// EnvGuestfsNetwork enables QEMU user networking in the appliance before
	// launch. Set to "1" or "true" when Clevis/NBDE unlock is required.
	EnvGuestfsNetwork = "KC_GUESTFS_NETWORK"
)

// StartSharedListener starts a cross-stage VM session via the named backend
// plugin (guestfs or qemu), attaching disks at boot. Returns an error when the
// backend does not support a shared listener.
func StartSharedListener(name string, disks []types.DiskSpec) (SharedListener, error) {
	plugin, err := backend.Resolve(name)
	if err != nil {
		return nil, err
	}
	slp, ok := plugin.(backend.SharedListenerPlugin)
	if !ok {
		return nil, fmt.Errorf("backend %s: shared listener not supported", name)
	}
	return slp.StartSharedListener(disks)
}

// SupportsSharedListener reports whether the named backend keeps a VM-resident
// session alive across pipeline stages. This is a capability query on the
// registered plugin type; it does not check runtime availability (that surfaces
// when StartSharedListener actually boots the session).
func SupportsSharedListener(name string) bool {
	plugin, err := backend.Lookup(name)
	if err != nil {
		return false
	}
	_, ok := plugin.(backend.SharedListenerPlugin)
	return ok
}

// SharedListenerAlive reports whether the listener's guestfish process is still running.
func SharedListenerAlive(l SharedListener) bool {
	return l != nil && l.Alive()
}
