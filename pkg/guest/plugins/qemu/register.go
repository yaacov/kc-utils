//go:build unix

package qemu

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest/backend"
)

func init() {
	backend.Factories.Register(string(backend.ModeQemu), &factory{})
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
	dir, err := os.MkdirTemp("", "kc-qemu-*")
	if err != nil {
		return nil, err
	}
	sock := filepath.Join(dir, "agent.sock")
	return &SharedSession{Sock: sock, dir: dir, owned: true}, nil
}

func (factory) PrepareClevisEnv() (func(), error) {
	prev, had := os.LookupEnv(backend.EnvGuestfsNetwork)
	if err := os.Setenv(backend.EnvGuestfsNetwork, "1"); err != nil {
		return nil, fmt.Errorf("enable qemu appliance network for Clevis: %w", err)
	}
	return func() {
		if !had {
			_ = os.Unsetenv(backend.EnvGuestfsNetwork)
			return
		}
		_ = os.Setenv(backend.EnvGuestfsNetwork, prev)
	}, nil
}
