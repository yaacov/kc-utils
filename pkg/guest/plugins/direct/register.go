//go:build unix

package direct

import (
	"fmt"
	"runtime"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest/backend"
)

func init() {
	if runtime.GOOS != "linux" {
		return
	}
	backend.Factories.Register(string(backend.ModeDirect), &factory{})
}

type factory struct{}

func (factory) Open(disks []types.DiskSpec, mountRoot string) (backend.Backend, error) {
	b := New()
	if err := b.Setup(disks, mountRoot); err != nil {
		_ = b.Teardown()
		return nil, fmt.Errorf("backend setup: %w", err)
	}
	return b, nil
}

func (factory) Attach(disks []types.DiskSpec, mountRoot string, infos []types.DiskInfo) (backend.Backend, error) {
	return NewMounted(disks, mountRoot, infos), nil
}

func (factory) TeardownMountRoot(mountRoot string) error {
	return TeardownMountRoot(mountRoot)
}
