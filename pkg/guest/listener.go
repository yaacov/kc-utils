//go:build linux

package guest

import "github.com/yaacov/kc-utils/pkg/guest/guestfs"

// SharedListener and StartSharedListener are re-exported from the guestfs
// subpackage so callers keep importing "pkg/guest" only.
type SharedListener = guestfs.SharedListener

var StartSharedListener = guestfs.StartSharedListener

// SharedListenerAlive reports whether the listener's guestfish process is
// still running.
func SharedListenerAlive(l *SharedListener) bool {
	return l.Alive()
}

const (
	EnvGuestfishPID   = guestfs.EnvGuestfishPID
	EnvKCGuestfishPID = guestfs.EnvKCGuestfishPID
)
