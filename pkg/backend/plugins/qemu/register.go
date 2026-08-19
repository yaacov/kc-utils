//go:build unix

package qemu

import (
	"github.com/yaacov/kc-utils/pkg/backend"
	"github.com/yaacov/kc-utils/pkg/common/types"
)

type factory struct{}

func init() {
	backend.Plugins.Register(backend.NameQEMU, &factory{})
}

func (f *factory) Name() string { return backend.NameQEMU }

// Requirements: a qemu-system binary must be present. Hardware acceleration is
// preferred but not required — the appliance falls back to TCG emulation — so
// Accel is deliberately not required here.
func (f *factory) Requirements() backend.Requirements {
	return backend.Requirements{QEMU: true}
}

// Available reports whether qemu is present and the appliance image for the
// host (or KC_APPLIANCE_ARCH) architecture is installed.
func (f *factory) Available() bool {
	if !backend.Available(f) {
		return false
	}
	if _, _, err := appliancePaths(applianceArch()); err != nil {
		return false
	}
	return true
}

func (f *factory) New() backend.Backend {
	return New()
}

func (f *factory) NewMounted(disks []types.DiskSpec, mountRoot string, infos []types.DiskInfo) (backend.Backend, error) {
	return NewMounted(disks, mountRoot, infos)
}

func (f *factory) TeardownMountRoot(mountRoot string) error {
	return TeardownMountRoot(mountRoot)
}

// StartSharedListener boots an appliance owned by the caller (kc-v2v) with the
// guest disks attached, so pipeline stages adopt it via KC_QEMU_SOCK.
func (f *factory) StartSharedListener(disks []types.DiskSpec) (backend.SharedListener, error) {
	return StartSharedListener(disks)
}

// sharedListener is a caller-owned appliance session shared across stages.
type sharedListener struct {
	session *vmSession
}

// StartSharedListener boots the appliance with disks attached and returns a
// handle the caller must Close.
func StartSharedListener(disks []types.DiskSpec) (*sharedListener, error) {
	session, err := newVMSession(toDriveSpecs(disks), clevisNetworkRequested())
	if err != nil {
		return nil, err
	}
	return &sharedListener{session: session}, nil
}

func (l *sharedListener) Env() []string {
	if l == nil || l.session == nil {
		return nil
	}
	return l.session.sharedEnv()
}

func (l *sharedListener) Alive() bool {
	return l != nil && l.session != nil && l.session.alive()
}

func (l *sharedListener) Close() error {
	if l == nil || l.session == nil {
		return nil
	}
	err := l.session.close()
	l.session = nil
	return err
}
