//go:build unix

package direct

import (
	"github.com/yaacov/kc-utils/pkg/backend"
	"github.com/yaacov/kc-utils/pkg/common/types"
)

type factory struct{}

func init() {
	backend.Plugins.Register(backend.NameDirect, &factory{})
}

func (f *factory) Name() string { return backend.NameDirect }

func (f *factory) Requirements() backend.Requirements {
	return backend.Requirements{Linux: true, Root: true}
}

func (f *factory) Available() bool {
	return backend.Available(f)
}

func (f *factory) New() backend.Backend {
	return New()
}

func (f *factory) NewMounted(disks []types.DiskSpec, mountRoot string, infos []types.DiskInfo) (backend.Backend, error) {
	return NewMounted(disks, mountRoot, infos), nil
}

func (f *factory) TeardownMountRoot(mountRoot string) error {
	return TeardownMountRoot(mountRoot)
}
