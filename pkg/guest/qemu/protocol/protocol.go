// Package protocol is the kc-agent RPC framing used by the qemu guest backend.
// It has no OS build tag so the host client (unix) and in-VM server (linux)
// can share types without importing each other.
package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

const (
	OpPing               = "ping"
	OpDiscover           = "discover"
	OpMount              = "mount"
	OpUnmountAll         = "unmount_all"
	OpProbeMount         = "probe_mount"
	OpProbeUnmount       = "probe_unmount"
	OpFSType             = "fstype"
	OpBlkidAttr          = "blkid_attr"
	OpFSCheck            = "fscheck"
	OpFSTrim             = "fstrim"
	OpDecrypt            = "decrypt"
	OpUnlockClevis       = "unlock_clevis"
	OpCloseCrypt         = "close_crypt"
	OpRescanBlock        = "rescan_block"
	OpRunCommand         = "run_command"
	OpDeviceRead         = "device_read"
	OpDeviceWrite        = "device_write"
	OpReadFile           = "read_file"
	OpWriteFile          = "write_file"
	OpExists             = "exists"
	OpIsDir              = "is_dir"
	OpGlob               = "glob"
	OpRemove             = "remove"
	OpRemoveAll          = "remove_all"
	OpRename             = "rename"
	OpSymlink            = "symlink"
	OpReadlink           = "readlink"
	OpChmod              = "chmod"
	OpMkdirAll           = "mkdir_all"
	OpReadDir            = "read_dir"
	OpUpload             = "upload"
	OpDownload           = "download"
	OpStatFS             = "statfs"
	OpSync               = "sync"
	OpUnmountFilesystems = "unmount_filesystems"
	OpReleaseDevices     = "release_devices"
	OpMergeHive          = "merge_hive"
	OpSetRoot            = "set_root"
)

const PortName = "org.kc-utils.agent"

// Request is a length-prefixed JSON RPC request. Binary payloads for
// upload/write follow the JSON frame when Size > 0.
type Request struct {
	ID   uint64          `json:"id"`
	Op   string          `json:"op"`
	Args json.RawMessage `json:"args,omitempty"`
	Size int64           `json:"size,omitempty"`
}

// Response is a length-prefixed JSON RPC response. Binary payloads for
// read/download follow when Size > 0.
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

type MountArgs struct {
	Device     string `json:"device"`
	MountPoint string `json:"mount_point"`
	FSType     string `json:"fstype"`
	ReadOnly   bool   `json:"read_only"`
}

type ProbeMountArgs struct {
	Device     string `json:"device"`
	FSType     string `json:"fstype"`
	MountPoint string `json:"mount_point"`
}

type BlkidArgs struct {
	Device string `json:"device"`
	Attr   string `json:"attr"`
}

type FSCheckArgs struct {
	Device string `json:"device"`
	FSType string `json:"fstype"`
}

type DecryptArgs struct {
	Device     string `json:"device"`
	MapperName string `json:"mapper_name"`
}

type UnlockClevisArgs struct {
	Device     string `json:"device"`
	MapperName string `json:"mapper_name"`
}

type RunCommandArgs struct {
	GuestRoot string   `json:"guest_root"`
	Cmd       []string `json:"cmd"`
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

type UploadArgs struct {
	GuestPath string `json:"guest_path"`
}

type MergeHiveArgs struct {
	Path string `json:"path"`
}

type StringResult struct {
	Value string `json:"value"`
}

type BoolResult struct {
	Value bool `json:"value"`
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

type StatFSResult struct {
	FreeBytes  int64 `json:"free_bytes"`
	FreeInodes int64 `json:"free_inodes"`
}

type DiscoverResult struct {
	Disks   []DiskInfo `json:"disks"`
	LVPaths []string   `json:"lv_paths"`
}

type DiskInfo struct {
	Serial     string          `json:"serial"`
	Device     string          `json:"device"`
	Partitions []PartitionInfo `json:"partitions"`
}

type PartitionInfo struct {
	Index      int    `json:"index"`
	DevicePath string `json:"device_path"`
	FSType     string `json:"fstype"`
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
