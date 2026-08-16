//go:build unix

// Package local implements runtime.Runtime with host syscalls and os/exec.
// It is used by the direct backend, where guest disks are loop-mounted on the
// host and every operation runs in the host's own namespace.
package local

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"

	guestcommon "github.com/yaacov/kc-utils/pkg/guest/common"
	"github.com/yaacov/kc-utils/pkg/guest/runtime"
)

// Runtime runs commands and file/device I/O directly on the host.
type Runtime struct{}

// New returns a host-local runtime.
func New() *Runtime { return &Runtime{} }

func (*Runtime) Run(spec *runtime.CommandSpec) (runtime.CommandResult, error) {
	if spec == nil || len(spec.Argv) == 0 {
		return runtime.CommandResult{Exit: -1}, nil
	}
	cmd := exec.Command(spec.Argv[0], spec.Argv[1:]...)
	cmd.Dir = spec.Dir
	if len(spec.Env) > 0 {
		cmd.Env = append(os.Environ(), spec.Env...)
	}
	if spec.Stdin != nil {
		cmd.Stdin = bytes.NewReader(spec.Stdin)
	}
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	res := runtime.CommandResult{Stdout: out.Bytes(), Stderr: errb.Bytes()}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			res.Exit = exitErr.ExitCode()
		} else {
			// Could not launch (binary missing, etc.): surface via Exit/Stderr,
			// not as a transport error, so callers handle it uniformly.
			res.Exit = -1
			res.Stderr = append(res.Stderr, []byte(err.Error())...)
		}
	}
	return res, nil
}

func (*Runtime) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }

func (*Runtime) WriteFile(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, perm)
}

func (*Runtime) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }
func (*Runtime) Remove(path string) error                     { return os.Remove(path) }
func (*Runtime) RemoveAll(path string) error                  { return os.RemoveAll(path) }
func (*Runtime) Rename(oldPath, newPath string) error         { return os.Rename(oldPath, newPath) }
func (*Runtime) Symlink(target, link string) error            { return os.Symlink(target, link) }
func (*Runtime) Readlink(path string) (string, error)         { return os.Readlink(path) }
func (*Runtime) Chmod(path string, mode os.FileMode) error    { return os.Chmod(path, mode) }

func (*Runtime) Stat(path string) (runtime.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return runtime.FileInfo{Exists: false}, nil
		}
		return runtime.FileInfo{}, err
	}
	return runtime.FileInfo{
		Exists: true,
		IsDir:  info.IsDir(),
		Mode:   info.Mode(),
		Size:   info.Size(),
	}, nil
}

func (*Runtime) ReadDir(path string) ([]runtime.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	out := make([]runtime.DirEntry, 0, len(entries))
	for _, e := range entries {
		mode := os.FileMode(0)
		if info, ierr := e.Info(); ierr == nil {
			mode = info.Mode()
		}
		out = append(out, runtime.DirEntry{Name: e.Name(), IsDir: e.IsDir(), Mode: mode})
	}
	return out, nil
}

func (*Runtime) Glob(pattern string) ([]string, error) { return filepath.Glob(pattern) }

func (*Runtime) DeviceRead(device string, offset int64, size int) ([]byte, error) {
	f, err := os.Open(device)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, size)
	if _, err := f.ReadAt(buf, offset); err != nil {
		return nil, err
	}
	return buf, nil
}

func (*Runtime) DeviceWrite(device string, offset int64, data []byte) error {
	f, err := os.OpenFile(device, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteAt(data, offset)
	return err
}

func (*Runtime) StatFS(path string) (int64, int64, error) {
	return guestcommon.HostStatFS(path)
}

func (*Runtime) Close() error { return nil }
