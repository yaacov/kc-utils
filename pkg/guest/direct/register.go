//go:build linux

package direct

import (
	"fmt"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest"
)

func init() {
	guest.Factories.Register(string(guest.ModeDirect), &factory{})
}

type factory struct{}

func (factory) Open(disks []types.DiskSpec, mountRoot string) (guest.Backend, error) {
	b := New()
	if err := b.Setup(disks, mountRoot); err != nil {
		_ = b.Teardown()
		return nil, fmt.Errorf("backend setup: %w", err)
	}
	return b, nil
}

func (factory) Attach(disks []types.DiskSpec, mountRoot string, infos []types.DiskInfo) (guest.Backend, error) {
	return NewMounted(disks, mountRoot, infos), nil
}

func (factory) TeardownMountRoot(mountRoot string) error {
	return TeardownMountRoot(mountRoot)
}
