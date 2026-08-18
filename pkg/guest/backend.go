//go:build unix

package guest

import (
	"github.com/yaacov/kc-utils/pkg/backend"
)

// Backend re-exports the backend interface from pkg/backend.
type Backend = backend.Backend

// DirEntry is a directory entry from ReadDir (guest paths).
type DirEntry = backend.DirEntry

// SharedListener is a cross-stage guestfish --listen session handle.
type SharedListener = backend.SharedListener

const (
	BackendDirect  = backend.NameDirect
	BackendGuestfs = backend.NameGuestfs
)
