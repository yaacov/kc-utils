//go:build unix

package guestfs

import (
	"fmt"
	"os"
	"runtime"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest/backend"
)

func init() {
	if runtime.GOOS != "linux" {
		return
	}
	backend.Factories.Register(string(backend.ModeGuestfs), &factory{})
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
	return NewMounted(disks, mountRoot, infos)
}

func (factory) TeardownMountRoot(mountRoot string) error {
	return TeardownMountRoot(mountRoot)
}

func (factory) StartSharedSession() (backend.SharedSession, error) {
	return StartSharedListener()
}

func (factory) PrepareClevisEnv() (func(), error) {
	prevNetwork, hadNetwork := os.LookupEnv(EnvGuestfsNetwork)
	if err := os.Setenv(EnvGuestfsNetwork, "1"); err != nil {
		return nil, fmt.Errorf("enable guestfs network for Clevis: %w", err)
	}
	return func() {
		if !hadNetwork {
			_ = os.Unsetenv(EnvGuestfsNetwork)
			return
		}
		_ = os.Setenv(EnvGuestfsNetwork, prevNetwork)
	}, nil
}
