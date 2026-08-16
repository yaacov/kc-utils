//go:build unix

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
	"github.com/yaacov/kc-utils/pkg/guest"
	"github.com/yaacov/kc-utils/pkg/guest/qemu/protocol"
)

const agentWait = 60 * time.Second

var (
	_ guest.Backend              = (*Backend)(nil)
	_ guest.SharedSessionFactory = factory{}
	_ guest.ClevisAwareFactory   = factory{}
)

// Backend talks to kc-agent in a QEMU appliance over a Unix socket.
type Backend struct {
	mountRoot string
	diskPaths []string
	diskInfos []types.DiskInfo
	lvPaths   []string
	client    *client
	cmd       *exec.Cmd
	sock      string
	owned     bool
}

func New() *Backend { return &Backend{} }

func NewMounted(disks []types.DiskSpec, mountRoot string, infos []types.DiskInfo) (*Backend, error) {
	b := New()
	b.mountRoot = mountRoot
	b.diskInfos = infos
	for _, d := range disks {
		b.diskPaths = append(b.diskPaths, d.Path)
	}
	if err := b.attachSession(); err != nil {
		return nil, err
	}
	return b, nil
}

func (b *Backend) attachSession() error {
	sock := agentSockFromEnv()
	if sock == "" {
		return fmt.Errorf("qemu attach requires %s", guest.EnvAgentSock)
	}
	cl, err := waitAgent(sock, agentWait)
	if err != nil {
		return err
	}
	b.client = cl
	b.sock = sock
	if b.mountRoot != "" {
		if err := b.client.call(protocol.OpSetRoot, protocol.PathArgs{Path: b.mountRoot}, nil); err != nil {
			return fmt.Errorf("set guest root %q: %w", b.mountRoot, err)
		}
	}
	return nil
}

func (b *Backend) Setup(disks []types.DiskSpec, mountRoot string) error {
	b.mountRoot = mountRoot
	for _, d := range disks {
		b.diskPaths = append(b.diskPaths, d.Path)
	}
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

	var disc protocol.DiscoverResult
	if err := b.client.call(protocol.OpDiscover, nil, &disc); err != nil {
		b.killOwned()
		return fmt.Errorf("appliance discover: %w", err)
	}
	if err := b.client.call(protocol.OpSetRoot, protocol.PathArgs{Path: mountRoot}, nil); err != nil {
		b.killOwned()
		return fmt.Errorf("set guest root %q: %w", mountRoot, err)
	}
	b.lvPaths = disc.LVPaths
	for i, d := range disks {
		di := types.DiskInfo{Path: d.Path, Format: d.Format}
		if i < len(disc.Disks) {
			for _, p := range disc.Disks[i].Partitions {
				di.Partitions = append(di.Partitions, types.PartitionInfo{
					Index:      p.Index,
					DevicePath: p.DevicePath,
					FSType:     p.FSType,
				})
			}
		}
		b.diskInfos = append(b.diskInfos, di)
	}
	slog.Info("qemu backend setup", "disks", len(disks), "sock", sock, "shared", !owned)
	return nil
}

func (b *Backend) DiskInfos() []types.DiskInfo { return b.diskInfos }
func (b *Backend) LVPaths() []string           { return b.lvPaths }
func (b *Backend) DiskPaths() []string         { return b.diskPaths }

func (b *Backend) Mount(device, hostMountPoint, fstype string, readOnly bool) error {
	return b.client.call(protocol.OpMount, protocol.MountArgs{
		Device: device, MountPoint: hostMountPoint, FSType: fstype, ReadOnly: readOnly,
	}, nil)
}

func (b *Backend) UnmountAll() error {
	return b.client.call(protocol.OpUnmountAll, nil, nil)
}

func (b *Backend) ProbeMount(device, fstype, hostMountPoint string) error {
	return b.client.call(protocol.OpProbeMount, protocol.ProbeMountArgs{
		Device: device, FSType: fstype, MountPoint: hostMountPoint,
	}, nil)
}

func (b *Backend) ProbeUnmount(hostMountPoint string) error {
	return b.client.call(protocol.OpProbeUnmount, protocol.PathArgs{Path: hostMountPoint}, nil)
}

func (b *Backend) FSType(device string) (string, error) {
	var r protocol.StringResult
	err := b.client.call(protocol.OpFSType, protocol.PathArgs{Path: device}, &r)
	return r.Value, err
}

func (b *Backend) BlkidAttr(device, attr string) (string, error) {
	var r protocol.StringResult
	err := b.client.call(protocol.OpBlkidAttr, protocol.BlkidArgs{Device: device, Attr: attr}, &r)
	return r.Value, err
}

func (b *Backend) FSCheck(device, fstype string) error {
	return b.client.call(protocol.OpFSCheck, protocol.FSCheckArgs{Device: device, FSType: fstype}, nil)
}

func (b *Backend) FSTrim(mountpoint string) error {
	return b.client.call(protocol.OpFSTrim, protocol.PathArgs{Path: mountpoint}, nil)
}

func (b *Backend) Decrypt(device, keyFile, mapperName string) (string, error) {
	var keyData []byte
	if keyFile != "" {
		data, err := os.ReadFile(keyFile)
		if err != nil {
			return "", err
		}
		keyData = data
	}
	var r protocol.StringResult
	_, err := b.client.callBlob(protocol.OpDecrypt, protocol.DecryptArgs{
		Device: device, MapperName: mapperName,
	}, keyData, &r)
	return r.Value, err
}

func (b *Backend) UnlockClevis(device, mapperName string) (string, error) {
	var r protocol.StringResult
	err := b.client.call(protocol.OpUnlockClevis, protocol.UnlockClevisArgs{
		Device: device, MapperName: mapperName,
	}, &r)
	return r.Value, err
}

func (b *Backend) CloseCrypt(mapperName string) error {
	return b.client.call(protocol.OpCloseCrypt, protocol.PathArgs{Path: mapperName}, nil)
}

func (b *Backend) RescanBlock() error {
	var disc protocol.DiscoverResult
	if err := b.client.call(protocol.OpRescanBlock, nil, &disc); err != nil {
		return err
	}
	b.lvPaths = disc.LVPaths
	return nil
}

func (b *Backend) RunCommand(guestRoot string, cmd []string) ([]byte, error) {
	return b.client.callBlob(protocol.OpRunCommand, protocol.RunCommandArgs{
		GuestRoot: guestRoot, Cmd: cmd,
	}, nil, nil)
}

func (b *Backend) DeviceRead(device string, offset int64, size int) ([]byte, error) {
	return b.client.callBlob(protocol.OpDeviceRead, protocol.DeviceRWArgs{
		Device: device, Offset: offset, Size: size,
	}, nil, nil)
}

func (b *Backend) DeviceWrite(device string, offset int64, data []byte) error {
	_, err := b.client.callBlob(protocol.OpDeviceWrite, protocol.DeviceRWArgs{
		Device: device, Offset: offset,
	}, data, nil)
	return err
}

func (b *Backend) ReadFile(guestPath string) ([]byte, error) {
	return b.client.readFile(guestPath)
}

func (b *Backend) WriteFile(guestPath string, data []byte, perm os.FileMode) error {
	return b.client.writeFile(guestPath, data, perm)
}

func (b *Backend) Exists(guestPath string) bool {
	var r protocol.BoolResult
	if err := b.client.call(protocol.OpExists, protocol.PathArgs{Path: guestPath}, &r); err != nil {
		return false
	}
	return r.Value
}

func (b *Backend) IsDir(guestPath string) bool {
	var r protocol.BoolResult
	if err := b.client.call(protocol.OpIsDir, protocol.PathArgs{Path: guestPath}, &r); err != nil {
		return false
	}
	return r.Value
}

func (b *Backend) Glob(pattern string) ([]string, error) {
	var r protocol.PathsResult
	err := b.client.call(protocol.OpGlob, protocol.PathArgs{Path: pattern}, &r)
	return r.Paths, err
}

func (b *Backend) Remove(guestPath string) error {
	return b.client.call(protocol.OpRemove, protocol.PathArgs{Path: guestPath}, nil)
}

func (b *Backend) RemoveAll(guestPath string) error {
	return b.client.call(protocol.OpRemoveAll, protocol.PathArgs{Path: guestPath}, nil)
}

func (b *Backend) Rename(oldPath, newPath string) error {
	return b.client.call(protocol.OpRename, protocol.RenameArgs{Old: oldPath, New: newPath}, nil)
}

func (b *Backend) Symlink(target, link string) error {
	return b.client.call(protocol.OpSymlink, protocol.SymlinkArgs{Target: target, Link: link}, nil)
}

func (b *Backend) Readlink(guestPath string) (string, error) {
	var r protocol.StringResult
	err := b.client.call(protocol.OpReadlink, protocol.PathArgs{Path: guestPath}, &r)
	return r.Value, err
}

func (b *Backend) Chmod(guestPath string, mode os.FileMode) error {
	return b.client.call(protocol.OpChmod, protocol.ChmodArgs{Path: guestPath, Mode: mode}, nil)
}

func (b *Backend) MkdirAll(guestPath string, perm os.FileMode) error {
	return b.client.call(protocol.OpMkdirAll, protocol.MkdirArgs{Path: guestPath, Perm: perm}, nil)
}

func (b *Backend) ReadDir(guestPath string) ([]guest.DirEntry, error) {
	var r protocol.DirResult
	if err := b.client.call(protocol.OpReadDir, protocol.PathArgs{Path: guestPath}, &r); err != nil {
		return nil, err
	}
	out := make([]guest.DirEntry, 0, len(r.Entries))
	for _, e := range r.Entries {
		out = append(out, guest.DirEntry{Name: e.Name, IsDir: e.IsDir, Mode: e.Mode})
	}
	return out, nil
}

func (b *Backend) Upload(hostPath, guestPath string) error {
	info, err := os.Stat(hostPath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return filepath.Walk(hostPath, func(p string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(hostPath, p)
			if err != nil {
				return err
			}
			gp := guestPath
			if rel != "." {
				gp = filepath.ToSlash(filepath.Join(guestPath, rel))
			}
			if fi.IsDir() {
				return b.MkdirAll(gp, 0o755)
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			return b.client.upload(gp, data)
		})
	}
	data, err := os.ReadFile(hostPath)
	if err != nil {
		return err
	}
	return b.client.upload(guestPath, data)
}

func (b *Backend) Download(guestPath, hostPath string) error {
	data, err := b.client.download(guestPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(hostPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(hostPath, data, 0o644)
}

func (b *Backend) StatFS(guestPath string) (int64, int64, error) {
	var r protocol.StatFSResult
	err := b.client.call(protocol.OpStatFS, protocol.PathArgs{Path: guestPath}, &r)
	return r.FreeBytes, r.FreeInodes, err
}

func (b *Backend) MergeHive(guestPath string, reg []byte) error {
	_, err := b.client.callBlob(protocol.OpMergeHive, protocol.MergeHiveArgs{Path: guestPath}, reg, nil)
	return err
}

func (b *Backend) Sync() error {
	return b.client.call(protocol.OpSync, nil, nil)
}

func (b *Backend) Release() error {
	if b.client != nil {
		_ = b.client.Close()
		b.client = nil
	}
	// Shared session: leave QEMU running for convert/finalize.
	return nil
}

func (b *Backend) UnmountFilesystems() error {
	return b.client.call(protocol.OpUnmountFilesystems, nil, nil)
}

func (b *Backend) ReleaseDevices() error {
	var err error
	if b.client != nil {
		err = b.client.call(protocol.OpReleaseDevices, nil, nil)
	}
	b.killOwned()
	return err
}

func (b *Backend) Teardown() error {
	var first error
	if b.client != nil {
		if err := b.UnmountFilesystems(); err != nil {
			first = err
		}
		if err := b.client.call(protocol.OpReleaseDevices, nil, nil); err != nil && first == nil {
			first = err
		}
	}
	b.killOwned()
	return first
}

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

func TeardownMountRoot(mountRoot string) error {
	if mountRoot == "" {
		return nil
	}
	return os.RemoveAll(mountRoot)
}
