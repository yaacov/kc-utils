//go:build linux

// Package server is the in-appliance kc-agent RPC implementation (Linux only).
package server

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/yaacov/kc-utils/pkg/guest/qemu/protocol"
)

const guestRootDefault = "/mnt/kc-guest"

// Agent serves Backend operations inside the appliance.
type Agent struct {
	guestRoot string
	cryptMaps []string
	mounts    []string
}

func New() *Agent {
	return &Agent{guestRoot: guestRootDefault}
}

func (a *Agent) host(guestPath string) string {
	p := filepath.Clean("/" + filepath.ToSlash(guestPath))
	if a.guestRoot != "" && (p == a.guestRoot || strings.HasPrefix(p, a.guestRoot+"/")) {
		return p
	}
	return filepath.Join(a.guestRoot, strings.TrimPrefix(p, "/"))
}

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
	case protocol.OpReadFile, protocol.OpDownload, protocol.OpDeviceRead, protocol.OpRunCommand:
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

func (a *Agent) mount(args protocol.MountArgs) error {
	if err := os.MkdirAll(args.MountPoint, 0o755); err != nil {
		return err
	}
	opts := "nodev,nosuid,noexec"
	if args.ReadOnly {
		opts += ",ro"
	}
	cmd := exec.Command("mount", "-t", args.FSType, "-o", opts, args.Device, args.MountPoint)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mount %s: %w\n%s", args.Device, err, out)
	}
	a.mounts = append(a.mounts, args.MountPoint)
	return nil
}

func (a *Agent) unmount(path string) error {
	err := exec.Command("umount", path).Run()
	filtered := a.mounts[:0]
	for _, m := range a.mounts {
		if m != path {
			filtered = append(filtered, m)
		}
	}
	a.mounts = filtered
	return err
}

func (a *Agent) unmountAll() {
	for i := len(a.mounts) - 1; i >= 0; i-- {
		_ = exec.Command("umount", a.mounts[i]).Run()
	}
	a.mounts = nil
}

func (a *Agent) decrypt(args protocol.DecryptArgs, key []byte) (string, error) {
	keyPath := ""
	if len(key) > 0 {
		tmp, err := os.CreateTemp("", "kc-luks-key-*")
		if err != nil {
			return "", err
		}
		keyPath = tmp.Name()
		defer os.Remove(keyPath)
		if _, err := tmp.Write(key); err != nil {
			tmp.Close()
			return "", err
		}
		tmp.Close()
	}
	cmdArgs := []string{"open", "--type", "luks", args.Device, args.MapperName}
	if keyPath != "" {
		cmdArgs = append(cmdArgs, "--key-file", keyPath)
	}
	if out, err := exec.Command("cryptsetup", cmdArgs...).CombinedOutput(); err != nil {
		return "", fmt.Errorf("cryptsetup: %w\n%s", err, out)
	}
	a.cryptMaps = append(a.cryptMaps, args.MapperName)
	return "/dev/mapper/" + args.MapperName, nil
}

func (a *Agent) releaseDevices() {
	for i := len(a.cryptMaps) - 1; i >= 0; i-- {
		_ = exec.Command("cryptsetup", "close", a.cryptMaps[i]).Run()
	}
	a.cryptMaps = nil
	_ = exec.Command("vgchange", "-an").Run()
}

func fscheck(device, fs string) error {
	switch fs {
	case "ext4", "ext3", "ext2":
		return exec.Command("e2fsck", "-f", "-y", device).Run()
	case "xfs":
		return exec.Command("xfs_repair", device).Run()
	case "btrfs":
		return exec.Command("btrfs", "check", device).Run()
	case "ntfs3":
		return exec.Command("ntfsfix", "-d", device).Run()
	default:
		return nil
	}
}

func detectFS(device string) (string, error) {
	f, err := os.Open(device)
	if err != nil {
		return "", err
	}
	defer f.Close()
	buf := make([]byte, 0x10100)
	// io.ReadFull fills the buffer so the btrfs magic at 0x10040 is reached; a
	// single Read may return early. Small devices legitimately return fewer
	// bytes, so a short read is not an error for detection purposes.
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return "", err
	}
	return detectMagic(buf[:n]), nil
}

func detectMagic(buf []byte) string {
	if len(buf) > 0x439 && buf[0x438] == 0x53 && buf[0x439] == 0xEF {
		return "ext4"
	}
	if len(buf) >= 4 && string(buf[0:4]) == "XFSB" {
		return "xfs"
	}
	if len(buf) > 0x10047 && string(buf[0x10040:0x10048]) == "_BHRfS_M" {
		return "btrfs"
	}
	if len(buf) >= 7 && string(buf[3:7]) == "NTFS" {
		return "ntfs3"
	}
	if len(buf) >= 0x58 && (string(buf[0x52:0x57]) == "FAT32" || string(buf[0x36:0x3B]) == "FAT16") {
		return "vfat"
	}
	return "unknown"
}

func (a *Agent) discover() []byte {
	_ = exec.Command("vgscan").Run()
	_ = exec.Command("vgchange", "-ay").Run()

	var disks []protocol.DiskInfo
	matches, _ := filepath.Glob("/dev/vd*")
	seen := map[string]bool{}
	for _, p := range matches {
		base := filepath.Base(p)
		if strings.ContainsAny(base[len(base)-1:], "0123456789") {
			continue
		}
		if seen[p] {
			continue
		}
		seen[p] = true
		di := protocol.DiskInfo{Serial: readSerial(base), Device: p}
		parts, _ := filepath.Glob(p + "[0-9]*")
		also, _ := filepath.Glob(p + "p[0-9]*")
		parts = append(parts, also...)
		for i, part := range parts {
			ft, _ := detectFS(part)
			di.Partitions = append(di.Partitions, protocol.PartitionInfo{
				Index: i + 1, DevicePath: part, FSType: ft,
			})
		}
		disks = append(disks, di)
	}
	var lvs []string
	if out, err := exec.Command("lvscan").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if !strings.Contains(line, "ACTIVE") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				lvs = append(lvs, strings.Trim(fields[1], "'"))
			}
		}
	}
	return marshal(protocol.DiscoverResult{Disks: disks, LVPaths: lvs})
}

func readSerial(dev string) string {
	b, err := os.ReadFile("/sys/block/" + dev + "/serial")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// Bootstrap prepares a pid-1 appliance environment and opens the agent port.
func Bootstrap() (io.ReadWriteCloser, error) {
	_ = os.MkdirAll("/proc", 0o755)
	_ = os.MkdirAll("/sys", 0o755)
	_ = os.MkdirAll("/dev", 0o755)
	_ = os.MkdirAll("/tmp", 0o755)
	_ = syscall.Mount("proc", "/proc", "proc", 0, "")
	_ = syscall.Mount("sysfs", "/sys", "sysfs", 0, "")
	_ = syscall.Mount("devtmpfs", "/dev", "devtmpfs", 0, "")

	mods := []string{
		"virtio_pci", "virtio_mmio", "virtio_blk", "virtio_console", "virtio_scsi",
		"ext4", "xfs", "btrfs", "fat", "vfat", "ntfs3",
		"dm_mod", "dm_crypt",
	}
	for _, m := range mods {
		_ = exec.Command("modprobe", m).Run()
	}

	port := "/dev/virtio-ports/" + protocol.PortName
	for i := 0; i < 50; i++ {
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
