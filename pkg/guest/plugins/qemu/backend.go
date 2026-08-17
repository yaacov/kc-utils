//go:build unix

// Package qemu runs guest disk operations inside a QEMU appliance VM. All
// domain logic lives in pkg/guest/core on top of a remote runtime that forwards
// primitive operations to kc-agent over RPC; this package adds only the parts
// that genuinely differ from direct: launching/attaching the appliance and
// mapping virtio-blk serials back to disk specs.
package qemu

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest/backend"
	"github.com/yaacov/kc-utils/pkg/guest/core"
)

const agentWait = 60 * time.Second

var (
	_ backend.Backend              = (*Backend)(nil)
	_ backend.SharedSessionFactory = factory{}
	_ backend.ClevisAwareFactory   = factory{}
)

// Backend talks to kc-agent in a QEMU appliance over a Unix socket. It embeds
// the shared core backend (running on a remote runtime) and owns only the
// appliance process lifecycle.
type Backend struct {
	*core.Backend
	mountRoot string
	client    *client
	sock      string
	cmd       *exec.Cmd
	owned     bool
}

func New() *Backend { return &Backend{} }

// NewMounted attaches to a running shared appliance session that already has the
// guest tree mounted (used by Attach for the convert/finalize handoff).
func NewMounted(disks []types.DiskSpec, mountRoot string, infos []types.DiskInfo) (*Backend, error) {
	b := New()
	b.mountRoot = mountRoot
	if err := b.attachSession(disks, infos); err != nil {
		return nil, err
	}
	return b, nil
}

func (b *Backend) attachSession(disks []types.DiskSpec, infos []types.DiskInfo) error {
	sock := agentSockFromEnv()
	if sock == "" {
		return fmt.Errorf("qemu attach requires %s", backend.EnvAgentSock)
	}
	cl, err := waitAgent(sock, agentWait)
	if err != nil {
		return err
	}
	b.client = cl
	b.sock = sock
	b.Backend = core.New(&remoteRuntime{c: cl}, false)
	b.SetGuestRoot(b.mountRoot)
	b.AdoptDisks(infos, diskSpecPaths(disks))
	return nil
}

// Setup launches (or attaches to) the appliance, then discovers partitions and
// LVM volumes via core, mapping each virtio-blk serial back to its disk spec.
func (b *Backend) Setup(disks []types.DiskSpec, mountRoot string) error {
	b.mountRoot = mountRoot

	sock := agentSockFromEnv()
	owned := false
	if sock == "" {
		dir, err := os.MkdirTemp("", "kc-qemu-*")
		if err != nil {
			return err
		}
		sock = filepath.Join(dir, "agent.sock")
		owned = true
	}
	b.sock = sock

	if _, err := os.Stat(sock); err != nil {
		cmd, err := startQEMU(disks, sock)
		if err != nil {
			return err
		}
		b.cmd = cmd
		b.owned = owned
	}

	cl, err := waitAgent(sock, agentWait)
	if err != nil {
		b.killOwned()
		return err
	}
	b.client = cl
	b.Backend = core.New(&remoteRuntime{c: cl}, false)
	b.SetGuestRoot(mountRoot)

	if err := b.discover(disks); err != nil {
		b.killOwned()
		return fmt.Errorf("appliance discover: %w", err)
	}
	slog.Info("qemu backend setup", "disks", len(disks), "sock", sock, "shared", !owned)
	return nil
}

// discover enumerates each disk's partitions in disk-spec order by matching the
// virtio-blk serial (kc-disk-<index>) assigned in BuildQEMUArgs, then activates
// LVM. Domain logic (lsblk parsing, LVM activation) lives entirely in core.
func (b *Backend) discover(disks []types.DiskSpec) error {
	blks, err := b.ListBlockDevices()
	if err != nil {
		return err
	}
	bySerial := make(map[string]string, len(blks))
	for _, d := range blks {
		if d.Serial != "" {
			bySerial[d.Serial] = d.Path
		}
	}
	for i, d := range disks {
		serial := fmt.Sprintf("kc-disk-%d", i)
		devPath, ok := bySerial[serial]
		if !ok {
			return fmt.Errorf("disk %d (%s): no appliance device with serial %s", i, d.Path, serial)
		}
		if err := b.DiscoverDevice(devPath, d.Path, d.Format); err != nil {
			return fmt.Errorf("discovering disk %s: %w", d.Path, err)
		}
	}
	if err := b.ScanLVM(); err != nil {
		slog.Warn("LVM scan failed", "error", err)
	}
	return nil
}

// ---- Lifecycle (appliance-specific; the rest is promoted from core) --------

// Release frees process-local state (probe mounts) and drops the client, while
// leaving QEMU and the guest mounts in place for the convert stage.
func (b *Backend) Release() error {
	var first error
	if b.Backend != nil {
		first = b.UnmountProbes()
	}
	if b.client != nil {
		_ = b.client.Close()
		b.client = nil
	}
	return first
}

// ReleaseDevices closes LUKS mappers and deactivates LVM in the appliance, then
// stops QEMU if this backend owns it.
func (b *Backend) ReleaseDevices() error {
	var first error
	if b.Backend != nil && b.client != nil {
		first = b.CloseCryptMaps()
		b.CloseAllCryptMaps()
		if err := b.DeactivateLVM(); err != nil && first == nil {
			first = err
		}
	}
	b.killOwned()
	return first
}

// Teardown unmounts every guest filesystem, releases devices, and stops QEMU.
func (b *Backend) Teardown() error {
	var first error
	if b.Backend != nil && b.client != nil {
		first = b.UnmountFilesystems()
	}
	if err := b.ReleaseDevices(); err != nil && first == nil {
		first = err
	}
	return first
}

// TeardownDiscard is identical to Teardown: the appliance never writes guest
// edits back unless the caller synced, so there is nothing extra to discard.
func (b *Backend) TeardownDiscard() error {
	return b.Teardown()
}

func (b *Backend) killOwned() {
	if b.client != nil {
		_ = b.client.Close()
		b.client = nil
	}
	if !b.owned || b.cmd == nil || b.cmd.Process == nil {
		return
	}
	_ = b.cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		_ = b.cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = b.cmd.Process.Kill()
		<-done
	}
	b.cmd = nil
	if b.sock != "" {
		_ = os.Remove(b.sock)
		_ = os.RemoveAll(filepath.Dir(b.sock))
	}
}

// TeardownMountRoot best-effort removes an orphaned mount root. The appliance
// holds no host mounts, so there is nothing to unmount host-side.
func TeardownMountRoot(mountRoot string) error {
	if mountRoot == "" {
		return nil
	}
	return os.RemoveAll(mountRoot)
}

func diskSpecPaths(disks []types.DiskSpec) []string {
	paths := make([]string, 0, len(disks))
	for _, d := range disks {
		paths = append(paths, d.Path)
	}
	return paths
}
