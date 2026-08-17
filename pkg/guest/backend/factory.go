//go:build unix

package backend

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/plugin"
	"github.com/yaacov/kc-utils/pkg/common/types"
)

// Factory creates and tears down a Backend implementation.
type Factory interface {
	Open(disks []types.DiskSpec, mountRoot string) (Backend, error)
	Attach(disks []types.DiskSpec, mountRoot string, infos []types.DiskInfo) (Backend, error)
	TeardownMountRoot(mountRoot string) error
}

// SharedSession is a long-lived backend session shared across pipeline stages
// (guestfs guestfish --listen today).
type SharedSession interface {
	Close() error
	Alive() bool
	Env() []string
}

// SharedSessionFactory is implemented by backends that need a cross-stage session.
type SharedSessionFactory interface {
	Factory
	StartSharedSession() (SharedSession, error)
}

// ClevisAwareFactory is implemented by backends that need prep before Clevis/NBDE unlock.
type ClevisAwareFactory interface {
	Factory
	// PrepareClevisEnv enables backend-specific networking (or equivalent).
	// The returned cleanup restores prior state.
	PrepareClevisEnv() (cleanup func(), err error)
}

// Factories holds registered guest disk backends, keyed by name ("direct", "guestfs", …).
var Factories = plugin.NewRegistry[string, Factory]()

// AvailableBackends returns registered backend names in sorted order.
func AvailableBackends() []string {
	names := Factories.List()
	sort.Strings(names)
	return names
}

// LookupFactory returns the factory for name, or an error listing available backends.
func LookupFactory(name string) (Factory, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		avail := AvailableBackends()
		if len(avail) == 0 {
			return nil, fmt.Errorf("backend is required (no backends registered)")
		}
		return nil, fmt.Errorf("backend is required (available: %s)", strings.Join(avail, ", "))
	}
	f, ok := Factories.Get(name)
	if !ok {
		avail := AvailableBackends()
		if len(avail) == 0 {
			return nil, fmt.Errorf("unknown backend %q (no backends registered)", name)
		}
		return nil, fmt.Errorf("unknown backend %q (available: %s)", name, strings.Join(avail, ", "))
	}
	return f, nil
}

// BackendFlagUsage returns help text listing registered backend names.
func BackendFlagUsage() string {
	names := AvailableBackends()
	if len(names) == 0 {
		return "guest disk backend name"
	}
	return "guest disk backend (" + strings.Join(names, "|") + ")"
}
