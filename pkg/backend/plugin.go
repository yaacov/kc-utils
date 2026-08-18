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

// SharedListenerPlugin is implemented by guestfs for cross-stage listener setup.
type SharedListenerPlugin interface {
	Plugin
	StartSharedListener() (SharedListener, error)
}

// Plugins is the global backend plugin registry.
var Plugins = plugin.NewRegistry[string, Plugin]()
