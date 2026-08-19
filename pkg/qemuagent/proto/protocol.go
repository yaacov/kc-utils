// Package proto defines the wire protocol between the kc-utils host qemu
// backend and the in-appliance kc-guest-agent.
//
// The appliance exposes only the primitive operations declared here (exec a
// program, file I/O, raw device I/O, stat/statfs). All higher-level conversion
// logic — partition discovery, LVM, LUKS, mount planning, fs-checks — is
// composed host-side out of these primitives.
//
// Messages are length-prefixed JSON frames: a 4-byte big-endian length followed
// by that many bytes of JSON. []byte fields are base64-encoded by encoding/json;
// this trades wire efficiency for readability and simplicity, which suits the
// modest payloads (config files, boot sectors, individual driver files).
package proto

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// Op identifies a primitive operation.
type Op string

const (
	OpPing      Op = "ping"
	OpExec      Op = "exec"
	OpReadFile  Op = "readfile"
	OpWriteFile Op = "writefile"
	OpStat      Op = "stat"
	OpReadDir   Op = "readdir"
	OpMkdir     Op = "mkdir"
	OpRemove    Op = "remove"
	OpRename    Op = "rename"
	OpSymlink   Op = "symlink"
	OpReadlink  Op = "readlink"
	OpChmod     Op = "chmod"
	OpPRead     Op = "pread"
	OpPWrite    Op = "pwrite"
	OpStatFS    Op = "statfs"
)

// MaxMessageSize caps a single frame to guard against corrupt length headers.
const MaxMessageSize = 512 << 20 // 512 MiB

// Request is a single primitive-operation request. Fields are op-specific and
// omitted when unused.
type Request struct {
	Op Op `json:"op"`

	// Exec
	Argv  []string `json:"argv,omitempty"`
	Stdin []byte   `json:"stdin,omitempty"`
	Env   []string `json:"env,omitempty"`

	// Path-based ops (ReadFile, WriteFile, Stat, ReadDir, Mkdir, Remove,
	// Chmod, Readlink, PRead, PWrite, StatFS)
	Path      string `json:"path,omitempty"`
	Data      []byte `json:"data,omitempty"`
	Mode      uint32 `json:"mode,omitempty"`
	Recursive bool   `json:"recursive,omitempty"`

	// Rename / Symlink
	OldPath string `json:"old_path,omitempty"`
	NewPath string `json:"new_path,omitempty"`
	Target  string `json:"target,omitempty"`
	Link    string `json:"link,omitempty"`

	// PRead / PWrite
	Offset int64 `json:"offset,omitempty"`
	Length int   `json:"length,omitempty"`
}

// DirEntry describes one directory entry returned by ReadDir.
type DirEntry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Mode  uint32 `json:"mode"`
	Size  int64  `json:"size"`
}

// Response is the reply to a Request. Err is non-empty when the operation
// itself failed at the transport/OS level; Exec reports program failure via
// ExitCode (and Stderr) with Err empty.
type Response struct {
	Err string `json:"err,omitempty"`

	// Exec
	Stdout   []byte `json:"stdout,omitempty"`
	Stderr   []byte `json:"stderr,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`

	// ReadFile / PRead
	Data []byte `json:"data,omitempty"`

	// Stat
	Exists bool   `json:"exists,omitempty"`
	IsDir  bool   `json:"is_dir,omitempty"`
	Mode   uint32 `json:"mode,omitempty"`
	Size   int64  `json:"size,omitempty"`

	// ReadDir
	Entries []DirEntry `json:"entries,omitempty"`

	// Readlink
	Target string `json:"target,omitempty"`

	// PWrite
	Written int `json:"written,omitempty"`

	// StatFS
	FreeBytes  int64 `json:"free_bytes,omitempty"`
	FreeInodes int64 `json:"free_inodes,omitempty"`
}

// WriteMsg writes v as a single length-prefixed JSON frame.
func WriteMsg(w io.Writer, v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	if len(payload) > MaxMessageSize {
		return fmt.Errorf("message too large: %d bytes", len(payload))
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(payload)))
	if _, err := w.Write(hdr[:]); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	if _, err := w.Write(payload); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}
	return nil
}

// ReadMsg reads one length-prefixed JSON frame into v. At a clean frame
// boundary it returns io.EOF so callers can detect connection close.
func ReadMsg(r io.Reader, v any) error {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return err // io.EOF at a boundary; ErrUnexpectedEOF mid-header
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n > MaxMessageSize {
		return fmt.Errorf("message too large: %d bytes", n)
	}
	payload := make([]byte, n)
	if _, err := io.ReadFull(r, payload); err != nil {
		return fmt.Errorf("read payload: %w", err)
	}
	if err := json.Unmarshal(payload, v); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}
	return nil
}
