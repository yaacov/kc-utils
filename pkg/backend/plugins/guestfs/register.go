//go:build unix

package guestfs

import (
	"github.com/yaacov/kc-utils/pkg/backend"
	"github.com/yaacov/kc-utils/pkg/common/types"
)

type factory struct{}

func init() {
	backend.Plugins.Register(backend.NameGuestfs, &factory{})
}

func (f *factory) Name() string { return backend.NameGuestfs }

func (f *factory) Requirements() backend.Requirements {
	return backend.Requirements{Linux: true, KVM: true, Guestfish: true}
}

func (f *factory) Available() bool {
	return backend.Available(f)
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

func (f *factory) StartSharedListener() (backend.SharedListener, error) {
	return StartSharedListener()
}
