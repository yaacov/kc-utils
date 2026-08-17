//go:build linux

package agent

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/yaacov/kc-utils/pkg/agent/protocol"
)

type handler func(*Agent, json.RawMessage, []byte) ([]byte, error)

var handlers = map[string]handler{
	protocol.OpPing:        (*Agent).ping,
	protocol.OpExec:        (*Agent).exec,
	protocol.OpReadFile:    (*Agent).readFile,
	protocol.OpWriteFile:   (*Agent).writeFile,
	protocol.OpMkdirAll:    (*Agent).mkdirAll,
	protocol.OpRemove:      (*Agent).remove,
	protocol.OpRemoveAll:   (*Agent).removeAll,
	protocol.OpRename:      (*Agent).rename,
	protocol.OpSymlink:     (*Agent).symlink,
	protocol.OpReadlink:    (*Agent).readlink,
	protocol.OpChmod:       (*Agent).chmod,
	protocol.OpStat:        (*Agent).stat,
	protocol.OpReadDir:     (*Agent).readDir,
	protocol.OpGlob:        (*Agent).glob,
	protocol.OpDeviceRead:  (*Agent).deviceRead,
	protocol.OpDeviceWrite: (*Agent).deviceWrite,
	protocol.OpStatFS:      (*Agent).statFS,
}

func (a *Agent) ping(_ json.RawMessage, _ []byte) ([]byte, error) {
	return marshal(protocol.StringResult{Value: "ok"}), nil
}

// exec runs a command and returns its output and exit code. A non-zero exit is
// reported in ExecResult.Exit, not as an RPC error, so the host-side core can
// interpret it (mirroring the local runtime's behaviour).
func (a *Agent) exec(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.ExecArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	res := protocol.ExecResult{Exit: -1}
	if len(args.Argv) == 0 {
		return marshal(res), nil
	}
	cmd := exec.Command(args.Argv[0], args.Argv[1:]...)
	cmd.Dir = args.Dir
	if len(args.Env) > 0 {
		cmd.Env = append(os.Environ(), args.Env...)
	}
	if args.Stdin != nil {
		cmd.Stdin = bytes.NewReader(args.Stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	res.Exit = 0
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			res.Exit = exitErr.ExitCode()
		} else {
			// Could not launch: surface via Exit/Stderr, not as a transport error.
			res.Exit = -1
			stderr.WriteString(err.Error())
		}
	}
	res.Stdout = stdout.Bytes()
	res.Stderr = stderr.Bytes()
	return marshal(res), nil
}

func (a *Agent) readFile(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	return os.ReadFile(args.Path)
}

func (a *Agent) writeFile(raw json.RawMessage, blob []byte) ([]byte, error) {
	var args protocol.WriteFileArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(args.Path), 0o755); err != nil {
		return nil, err
	}
	perm := args.Perm
	if perm == 0 {
		perm = 0o644
	}
	return nil, os.WriteFile(args.Path, blob, perm)
}

func (a *Agent) mkdirAll(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.MkdirArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	perm := args.Perm
	if perm == 0 {
		perm = 0o755
	}
	return nil, os.MkdirAll(args.Path, perm)
}

func (a *Agent) remove(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	return nil, os.Remove(args.Path)
}

func (a *Agent) removeAll(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	return nil, os.RemoveAll(args.Path)
}

func (a *Agent) rename(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.RenameArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	return nil, os.Rename(args.Old, args.New)
}

func (a *Agent) symlink(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.SymlinkArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	return nil, os.Symlink(args.Target, args.Link)
}

func (a *Agent) readlink(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	v, err := os.Readlink(args.Path)
	return marshal(protocol.StringResult{Value: v}), err
}

func (a *Agent) chmod(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.ChmodArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	return nil, os.Chmod(args.Path, args.Mode)
}

func (a *Agent) stat(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	info, err := os.Stat(args.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return marshal(protocol.StatResult{Exists: false}), nil
		}
		return nil, err
	}
	return marshal(protocol.StatResult{
		Exists: true,
		IsDir:  info.IsDir(),
		Mode:   info.Mode(),
		Size:   info.Size(),
	}), nil
}

func (a *Agent) readDir(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(args.Path)
	if err != nil {
		return nil, err
	}
	out := make([]protocol.DirEntry, 0, len(entries))
	for _, e := range entries {
		mode := os.FileMode(0)
		if info, ierr := e.Info(); ierr == nil {
			mode = info.Mode()
		}
		out = append(out, protocol.DirEntry{Name: e.Name(), IsDir: e.IsDir(), Mode: mode})
	}
	return marshal(protocol.DirResult{Entries: out}), nil
}

func (a *Agent) glob(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	matches, err := filepath.Glob(args.Path)
	if err != nil {
		return nil, err
	}
	return marshal(protocol.PathsResult{Paths: matches}), nil
}

func (a *Agent) deviceRead(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.DeviceRWArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	f, err := os.Open(args.Device)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, args.Size)
	if _, err := f.ReadAt(buf, args.Offset); err != nil {
		return nil, err
	}
	return buf, nil
}

func (a *Agent) deviceWrite(raw json.RawMessage, blob []byte) ([]byte, error) {
	var args protocol.DeviceRWArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(args.Device, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	_, err = f.WriteAt(blob, args.Offset)
	return nil, err
}

func (a *Agent) statFS(raw json.RawMessage, _ []byte) ([]byte, error) {
	var args protocol.PathArgs
	if err := decode(raw, &args); err != nil {
		return nil, err
	}
	var st syscall.Statfs_t
	if err := syscall.Statfs(args.Path, &st); err != nil {
		return nil, err
	}
	ffree := int64(st.Ffree)
	if st.Files == 0 && st.Ffree == 0 {
		ffree = -1
	}
	return marshal(protocol.StatFSResult{
		FreeBytes:  int64(st.Bavail) * int64(st.Bsize),
		FreeInodes: ffree,
	}), nil
}
