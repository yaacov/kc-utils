//go:build unix

// Package runtime abstracts command execution and file/device I/O so guest
// disk backends can run the same operation logic either on the host (direct)
// or inside a QEMU appliance over RPC (qemu).
//
// A Runtime is a thin, domain-free transport: it knows how to run a command
// and read/write files and devices. All domain logic (which tool to run, how
// to interpret its output, path translation) lives in pkg/guest/core on top
// of a Runtime — never in the Runtime itself.
package runtime

import (
	"os"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

// CommandSpec describes a command to run. Argv is passed directly to exec
// (no shell), so callers must not rely on shell quoting or globbing.
type CommandSpec struct {
	Argv  []string
	Dir   string   // working directory ("" = inherit)
	Env   []string // extra environment appended to the process env
	Stdin []byte   // optional standard input
}

// CommandResult captures the outcome of a command. Exit is the process exit
// code (-1 when the command could not be launched at all). A non-zero Exit is
// not itself a transport error; Run returns a non-nil error only when the
// command could not be dispatched (e.g. a broken RPC connection).
type CommandResult struct {
	Stdout []byte
	Stderr []byte
	Exit   int
}

// Combined returns stdout followed by stderr, mirroring exec CombinedOutput.
func (r CommandResult) Combined() []byte {
	out := make([]byte, 0, len(r.Stdout)+len(r.Stderr))
	out = append(out, r.Stdout...)
	out = append(out, r.Stderr...)
	return out
}

// FileInfo is a minimal, transport-friendly stat result.
type FileInfo struct {
	Exists bool
	IsDir  bool
	Mode   os.FileMode
	Size   int64
}

// DirEntry is a directory entry (reused from the shared types package).
type DirEntry = types.GuestDirEntry

// Runtime executes commands and performs file/device I/O in one location —
// the host for the local runtime, the appliance for the remote runtime. All
// paths are absolute in that location's own namespace (no guest-root prefixing).
type Runtime interface {
	// Run executes spec.Argv and returns its output and exit code.
	Run(spec *CommandSpec) (CommandResult, error)

	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
	Remove(path string) error
	RemoveAll(path string) error
	Rename(oldPath, newPath string) error
	Symlink(target, link string) error
	Readlink(path string) (string, error)
	Chmod(path string, mode os.FileMode) error
	Stat(path string) (FileInfo, error)
	ReadDir(path string) ([]DirEntry, error)
	Glob(pattern string) ([]string, error)

	// DeviceRead/DeviceWrite access raw bytes of a block device.
	DeviceRead(device string, offset int64, size int) ([]byte, error)
	DeviceWrite(device string, offset int64, data []byte) error

	// StatFS returns free bytes and free inodes for the filesystem at path.
	StatFS(path string) (freeBytes, freeInodes int64, err error)

	// Close releases any transport resources (a no-op for the local runtime).
	Close() error
}
