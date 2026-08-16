//go:build linux

// Package agent is the in-appliance kc-agent RPC implementation (Linux only).
//
// The agent is a generic runtime: it runs commands and performs raw file/device
// I/O on absolute paths in its own namespace. It holds no domain state (no
// mount/crypt/LVM bookkeeping) — the host-side core drives it by running
// standard tools (lsblk, mount, cryptsetup, ...) over OpExec.
package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/yaacov/kc-utils/pkg/agent/protocol"
)

// Agent serves primitive runtime operations inside the appliance. It is
// stateless across requests.
type Agent struct{}

func New() *Agent { return &Agent{} }

// Serve reads requests from rw until EOF.
func (a *Agent) Serve(rw io.ReadWriter) error {
	for {
		var req protocol.Request
		if err := protocol.ReadFrame(rw, &req); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		var blob []byte
		if req.Size > 0 {
			b, err := protocol.ReadBlob(rw)
			if err != nil {
				return err
			}
			blob = b
		}
		resp, out := a.handle(req, blob)
		if err := protocol.WriteFrame(rw, resp); err != nil {
			return err
		}
		if resp.Size > 0 && len(out) > 0 {
			if err := protocol.WriteBlob(rw, out); err != nil {
				return err
			}
		}
	}
}

func (a *Agent) handle(req protocol.Request, blob []byte) (protocol.Response, []byte) {
	resp := protocol.Response{ID: req.ID, OK: true}
	out, err := a.dispatch(req.Op, req.Args, blob)
	if err != nil {
		resp.OK = false
		resp.Error = err.Error()
		return resp, nil
	}
	switch req.Op {
	case protocol.OpReadFile, protocol.OpDeviceRead:
		resp.Size = int64(len(out))
		return resp, out
	}
	if len(out) > 0 {
		resp.Result = out
	}
	return resp, nil
}

func marshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func decode(raw json.RawMessage, v any) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, v)
}

func (a *Agent) dispatch(op string, raw json.RawMessage, blob []byte) ([]byte, error) {
	h, ok := handlers[op]
	if !ok {
		return nil, fmt.Errorf("unknown op %q", op)
	}
	return h(a, raw, blob)
}

// Bootstrap prepares a pid-1 appliance environment and opens the agent port.
func Bootstrap() (io.ReadWriteCloser, error) {
	_ = os.MkdirAll("/proc", 0o755)
	_ = os.MkdirAll("/sys", 0o755)
	_ = os.MkdirAll("/dev", 0o755)
	_ = os.MkdirAll("/tmp", 0o755)
	_ = mount("proc", "/proc", "proc")
	_ = mount("sysfs", "/sys", "sysfs")
	_ = mount("devtmpfs", "/dev", "devtmpfs")

	mods := []string{
		"virtio_pci", "virtio_mmio", "virtio_blk", "virtio_console", "virtio_scsi",
		"ext4", "xfs", "btrfs", "fat", "vfat", "ntfs3",
		"dm_mod", "dm_crypt",
	}
	for _, m := range mods {
		_ = exec.Command("modprobe", m).Run()
	}

	port := "/dev/virtio-ports/" + protocol.PortName
	for range 50 {
		f, err := os.OpenFile(port, os.O_RDWR, 0)
		if err == nil {
			return f, nil
		}
		if _, waitErr := os.Stat(port); waitErr == nil {
			return nil, err
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("agent port %s not found", port)
}

// mount is a thin wrapper over mount(2) for the pid-1 pseudo-filesystems.
func mount(source, target, fstype string) error {
	return syscall.Mount(source, target, fstype, 0, "")
}
