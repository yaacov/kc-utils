//go:build unix

package qemu

import (
	"os"

	"github.com/yaacov/kc-utils/pkg/agent/protocol"
	"github.com/yaacov/kc-utils/pkg/guest/runtime"
)

// remoteRuntime implements runtime.Runtime by forwarding every primitive
// operation to kc-agent over the RPC client. It carries no domain logic — the
// host-side core drives it exactly as it drives the local runtime, so all paths
// are absolute in the appliance's own namespace.
type remoteRuntime struct {
	c *client
}

var _ runtime.Runtime = (*remoteRuntime)(nil)

func (r *remoteRuntime) Run(spec *runtime.CommandSpec) (runtime.CommandResult, error) {
	var res protocol.ExecResult
	if err := r.c.call(protocol.OpExec, protocol.ExecArgs{
		Argv:  spec.Argv,
		Dir:   spec.Dir,
		Env:   spec.Env,
		Stdin: spec.Stdin,
	}, &res); err != nil {
		return runtime.CommandResult{Exit: -1}, err
	}
	return runtime.CommandResult{
		Stdout: res.Stdout,
		Stderr: res.Stderr,
		Exit:   res.Exit,
	}, nil
}

func (r *remoteRuntime) ReadFile(path string) ([]byte, error) {
	return r.c.callBlob(protocol.OpReadFile, protocol.PathArgs{Path: path}, nil, nil)
}

func (r *remoteRuntime) WriteFile(path string, data []byte, perm os.FileMode) error {
	_, err := r.c.callBlob(protocol.OpWriteFile, protocol.WriteFileArgs{Path: path, Perm: perm}, data, nil)
	return err
}

func (r *remoteRuntime) MkdirAll(path string, perm os.FileMode) error {
	return r.c.call(protocol.OpMkdirAll, protocol.MkdirArgs{Path: path, Perm: perm}, nil)
}

func (r *remoteRuntime) Remove(path string) error {
	return r.c.call(protocol.OpRemove, protocol.PathArgs{Path: path}, nil)
}

func (r *remoteRuntime) RemoveAll(path string) error {
	return r.c.call(protocol.OpRemoveAll, protocol.PathArgs{Path: path}, nil)
}

func (r *remoteRuntime) Rename(oldPath, newPath string) error {
	return r.c.call(protocol.OpRename, protocol.RenameArgs{Old: oldPath, New: newPath}, nil)
}

func (r *remoteRuntime) Symlink(target, link string) error {
	return r.c.call(protocol.OpSymlink, protocol.SymlinkArgs{Target: target, Link: link}, nil)
}

func (r *remoteRuntime) Readlink(path string) (string, error) {
	var res protocol.StringResult
	err := r.c.call(protocol.OpReadlink, protocol.PathArgs{Path: path}, &res)
	return res.Value, err
}

func (r *remoteRuntime) Chmod(path string, mode os.FileMode) error {
	return r.c.call(protocol.OpChmod, protocol.ChmodArgs{Path: path, Mode: mode}, nil)
}

func (r *remoteRuntime) Stat(path string) (runtime.FileInfo, error) {
	var res protocol.StatResult
	if err := r.c.call(protocol.OpStat, protocol.PathArgs{Path: path}, &res); err != nil {
		return runtime.FileInfo{}, err
	}
	return runtime.FileInfo{
		Exists: res.Exists,
		IsDir:  res.IsDir,
		Mode:   res.Mode,
		Size:   res.Size,
	}, nil
}

func (r *remoteRuntime) ReadDir(path string) ([]runtime.DirEntry, error) {
	var res protocol.DirResult
	if err := r.c.call(protocol.OpReadDir, protocol.PathArgs{Path: path}, &res); err != nil {
		return nil, err
	}
	out := make([]runtime.DirEntry, 0, len(res.Entries))
	for _, e := range res.Entries {
		out = append(out, runtime.DirEntry{Name: e.Name, IsDir: e.IsDir, Mode: e.Mode})
	}
	return out, nil
}

func (r *remoteRuntime) Glob(pattern string) ([]string, error) {
	var res protocol.PathsResult
	err := r.c.call(protocol.OpGlob, protocol.PathArgs{Path: pattern}, &res)
	return res.Paths, err
}

func (r *remoteRuntime) DeviceRead(device string, offset int64, size int) ([]byte, error) {
	return r.c.callBlob(protocol.OpDeviceRead, protocol.DeviceRWArgs{
		Device: device, Offset: offset, Size: size,
	}, nil, nil)
}

func (r *remoteRuntime) DeviceWrite(device string, offset int64, data []byte) error {
	_, err := r.c.callBlob(protocol.OpDeviceWrite, protocol.DeviceRWArgs{
		Device: device, Offset: offset,
	}, data, nil)
	return err
}

func (r *remoteRuntime) StatFS(path string) (int64, int64, error) {
	var res protocol.StatFSResult
	err := r.c.call(protocol.OpStatFS, protocol.PathArgs{Path: path}, &res)
	return res.FreeBytes, res.FreeInodes, err
}

func (r *remoteRuntime) Close() error {
	return r.c.Close()
}
