//go:build linux

package guest

import (
	"fmt"

	"github.com/yaacov/kc-utils/pkg/backend"
)

const (
	EnvGuestfishPID   = "GUESTFISH_PID"
	EnvKCGuestfishPID = "KC_GUESTFISH_PID"
	// EnvGuestfsNetwork enables QEMU user networking in the appliance before
	// launch. Set to "1" or "true" when Clevis/NBDE unlock is required.
	EnvGuestfsNetwork = "KC_GUESTFS_NETWORK"
)

// StartSharedListener starts a guestfish --listen session via the guestfs backend plugin.
func StartSharedListener() (SharedListener, error) {
	plugin, err := backend.Resolve(BackendGuestfs)
	if err != nil {
		return nil, err
	}
	slp, ok := plugin.(backend.SharedListenerPlugin)
	if !ok {
		return nil, fmt.Errorf("backend %s: shared listener not supported", BackendGuestfs)
	}
	return slp.StartSharedListener()
}

// SharedListenerAlive reports whether the listener's guestfish process is still running.
func SharedListenerAlive(l SharedListener) bool {
	return l != nil && l.Alive()
}
