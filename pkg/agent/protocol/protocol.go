// Package protocol is the kc-agent RPC framing used by the qemu guest backend.
// It has no OS build tag so the host client (unix) and in-VM server (linux)
// can share types without importing each other.
//
// The agent is a generic runtime: it exposes only primitive operations (run a
// command, read/write files and devices, stat). All domain logic (mount,
// decrypt, discover, fsck, ...) lives host-side in pkg/guest/core, which drives
// the agent by running standard tools over OpExec.
package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

const (
	OpPing        = "ping"
	OpExec        = "exec"
	OpReadFile    = "read_file"
	OpWriteFile   = "write_file"
	OpMkdirAll    = "mkdir_all"
	OpRemove      = "remove"
	OpRemoveAll   = "remove_all"
	OpRename      = "rename"
	OpSymlink     = "symlink"
	OpReadlink    = "readlink"
	OpChmod       = "chmod"
	OpStat        = "stat"
	OpReadDir     = "read_dir"
	OpGlob        = "glob"
	OpDeviceRead  = "device_read"
	OpDeviceWrite = "device_write"
	OpStatFS      = "statfs"
)

const PortName = "org.kc-utils.agent"

// Request is a length-prefixed JSON RPC request. A binary payload (write/device
// write) follows the JSON frame when Size > 0.
type Request struct {
	ID   uint64          `json:"id"`
	Op   string          `json:"op"`
	Args json.RawMessage `json:"args,omitempty"`
	Size int64           `json:"size,omitempty"`
}

// Response is a length-prefixed JSON RPC response. A binary payload (read/device
// read) follows when Size > 0.
type Response struct {
	ID     uint64          `json:"id"`
	OK     bool            `json:"ok"`
	Error  string          `json:"error,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Size   int64           `json:"size,omitempty"`
}

type PathArgs struct {
	Path string `json:"path"`
}

// ExecArgs runs Argv[0] with Argv[1:]. Stdin is small inline input; large
// output is returned in ExecResult (also inline — command output is bounded).
type ExecArgs struct {
	Argv  []string `json:"argv"`
	Dir   string   `json:"dir,omitempty"`
	Env   []string `json:"env,omitempty"`
	Stdin []byte   `json:"stdin,omitempty"`
}

// ExecResult reports command output. Exit is the process exit code (-1 if the
// command could not be launched). A non-zero Exit is not an RPC-level error.
type ExecResult struct {
	Stdout []byte `json:"stdout,omitempty"`
	Stderr []byte `json:"stderr,omitempty"`
	Exit   int    `json:"exit"`
}

type DeviceRWArgs struct {
	Device string `json:"device"`
	Offset int64  `json:"offset"`
	Size   int    `json:"size,omitempty"`
}

type WriteFileArgs struct {
	Path string      `json:"path"`
	Perm os.FileMode `json:"perm"`
}

type RenameArgs struct {
	Old string `json:"old"`
	New string `json:"new"`
}

type SymlinkArgs struct {
	Target string `json:"target"`
	Link   string `json:"link"`
}

type ChmodArgs struct {
	Path string      `json:"path"`
	Mode os.FileMode `json:"mode"`
}

type MkdirArgs struct {
	Path string      `json:"path"`
	Perm os.FileMode `json:"perm"`
}

type StringResult struct {
	Value string `json:"value"`
}

type PathsResult struct {
	Paths []string `json:"paths"`
}

type DirEntry struct {
	Name  string      `json:"name"`
	IsDir bool        `json:"is_dir"`
	Mode  os.FileMode `json:"mode"`
}

type DirResult struct {
	Entries []DirEntry `json:"entries"`
}

type StatResult struct {
	Exists bool        `json:"exists"`
	IsDir  bool        `json:"is_dir"`
	Mode   os.FileMode `json:"mode"`
	Size   int64       `json:"size"`
}

type StatFSResult struct {
	FreeBytes  int64 `json:"free_bytes"`
	FreeInodes int64 `json:"free_inodes"`
}

const maxFrame = 32 << 20 // 32 MiB JSON cap

// WriteFrame writes a 4-byte big-endian length prefix plus JSON.
func WriteFrame(w io.Writer, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if len(data) > maxFrame {
		return fmt.Errorf("rpc frame too large: %d", len(data))
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// ReadFrame reads a length-prefixed JSON value.
func ReadFrame(r io.Reader, v any) error {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n == 0 || n > maxFrame {
		return fmt.Errorf("invalid rpc frame size %d", n)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	return json.Unmarshal(buf, v)
}

// WriteBlob writes an 8-byte big-endian length plus raw bytes.
func WriteBlob(w io.Writer, data []byte) error {
	var hdr [8]byte
	binary.BigEndian.PutUint64(hdr[:], uint64(len(data)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

// ReadBlob reads an 8-byte length plus raw bytes.
func ReadBlob(r io.Reader) ([]byte, error) {
	var hdr [8]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint64(hdr[:])
	if n > 1<<32 {
		return nil, fmt.Errorf("rpc blob too large: %d", n)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
