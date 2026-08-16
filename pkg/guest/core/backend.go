//go:build unix

// Package core implements the guest disk operation logic shared by the direct
// and qemu backends. It is written entirely on top of a runtime.Runtime, so
// the same code runs on the host (direct, local runtime) or inside a QEMU
// appliance over RPC (qemu, remote runtime).
//
// All logic prefers standard util-linux/LVM/cryptsetup tools (lsblk, blkid,
// mount, cryptsetup, pvscan/vgscan, e2fsck, ...) over hand-rolled equivalents.
// Backends layer only the parts that genuinely differ — device attachment
// (loop devices vs virtio-blk) and process lifecycle — on top of this Backend.
package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest/runtime"
)

// Backend holds the runtime and the discovered/tracked state common to every
// non-guestfs backend. Embed it in a concrete backend and add Setup/Teardown.
type Backend struct {
	rt        runtime.Runtime
	guestRoot string
	live      bool // true when rt paths are host-visible (direct)

	mounts    []string // active guest filesystem mounts (deepest last)
	probes    []string // temporary read-only probe mounts
	cryptMaps []string // open LUKS mapper names

	partDevices []string // all partition device paths, for LVM rescans
	diskInfos   []types.DiskInfo
	lvPaths     []string
	diskPaths   []string
}

// New returns a core backend bound to rt. guestRoot is the absolute mount root
// in rt's namespace; live reports whether that root is directly host-visible.
func New(rt runtime.Runtime, live bool) *Backend {
	return &Backend{rt: rt, live: live}
}

// Runtime returns the underlying runtime (used by backends for lifecycle work).
func (b *Backend) Runtime() runtime.Runtime { return b.rt }

// SetGuestRoot records the mount root used to translate guest paths.
func (b *Backend) SetGuestRoot(root string) { b.guestRoot = root }

// Live reports whether guest paths map to host-visible paths.
func (b *Backend) Live() bool { return b.live }

func (b *Backend) DiskInfos() []types.DiskInfo { return b.diskInfos }
func (b *Backend) LVPaths() []string           { return b.lvPaths }
func (b *Backend) DiskPaths() []string         { return b.diskPaths }

// AdoptDisks seeds previously-discovered disk metadata without re-scanning
// (used when attaching to an already-prepared mount tree).
func (b *Backend) AdoptDisks(diskInfos []types.DiskInfo, diskPaths []string) {
	b.diskInfos = diskInfos
	b.diskPaths = diskPaths
}

// host translates a guest-absolute path (e.g. /etc/fstab) to an absolute path
// in the runtime's namespace under guestRoot.
func (b *Backend) host(guestPath string) string {
	p := filepath.Clean("/" + filepath.ToSlash(guestPath))
	if b.guestRoot != "" && (p == b.guestRoot || strings.HasPrefix(p, b.guestRoot+"/")) {
		return p
	}
	return filepath.Join(b.guestRoot, strings.TrimPrefix(p, "/"))
}

// HostPath exposes the runtime-namespace path for guestPath (used by backends
// that can hand host-visible paths back to callers).
func (b *Backend) HostPath(guestPath string) string { return b.host(guestPath) }

// run executes argv and returns an error if it could not be dispatched or
// exited non-zero (with combined output for context).
func (b *Backend) run(argv ...string) (runtime.CommandResult, error) {
	res, err := b.rt.Run(&runtime.CommandSpec{Argv: argv})
	if err != nil {
		return res, fmt.Errorf("%s: %w", argv[0], err)
	}
	if res.Exit != 0 {
		return res, fmt.Errorf("%s: exit %d: %s", argv[0], res.Exit, strings.TrimSpace(string(res.Combined())))
	}
	return res, nil
}

// runStatus dispatches argv and returns the result without erroring on a
// non-zero exit; it errors only when the command could not be run at all.
// Callers that must interpret specific exit codes (e.g. e2fsck) use this.
func (b *Backend) runStatus(argv ...string) (runtime.CommandResult, error) {
	res, err := b.rt.Run(&runtime.CommandSpec{Argv: argv})
	if err != nil {
		return res, fmt.Errorf("%s: %w", argv[0], err)
	}
	return res, nil
}

// ---- Raw device access -------------------------------------------------

func (b *Backend) DeviceRead(device string, offset int64, size int) ([]byte, error) {
	return b.rt.DeviceRead(device, offset, size)
}

func (b *Backend) DeviceWrite(device string, offset int64, data []byte) error {
	return b.rt.DeviceWrite(device, offset, data)
}

// ---- Guest filesystem operations (guest-absolute paths) ----------------

func (b *Backend) ReadFile(guestPath string) ([]byte, error) {
	return b.rt.ReadFile(b.host(guestPath))
}

func (b *Backend) WriteFile(guestPath string, data []byte, perm os.FileMode) error {
	if perm == 0 {
		perm = 0o644
	}
	return b.rt.WriteFile(b.host(guestPath), data, perm)
}

func (b *Backend) Exists(guestPath string) bool {
	info, err := b.rt.Stat(b.host(guestPath))
	return err == nil && info.Exists
}

func (b *Backend) IsDir(guestPath string) bool {
	info, err := b.rt.Stat(b.host(guestPath))
	return err == nil && info.Exists && info.IsDir
}

func (b *Backend) Glob(pattern string) ([]string, error) {
	matches, err := b.rt.Glob(b.host(pattern))
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		rel, err := filepath.Rel(b.guestRoot, m)
		if err != nil {
			continue
		}
		out = append(out, "/"+filepath.ToSlash(rel))
	}
	return out, nil
}

func (b *Backend) Remove(guestPath string) error    { return b.rt.Remove(b.host(guestPath)) }
func (b *Backend) RemoveAll(guestPath string) error { return b.rt.RemoveAll(b.host(guestPath)) }

func (b *Backend) Rename(oldPath, newPath string) error {
	return b.rt.Rename(b.host(oldPath), b.host(newPath))
}

func (b *Backend) Symlink(target, link string) error { return b.rt.Symlink(target, b.host(link)) }

func (b *Backend) Readlink(guestPath string) (string, error) {
	return b.rt.Readlink(b.host(guestPath))
}

func (b *Backend) Chmod(guestPath string, mode os.FileMode) error {
	return b.rt.Chmod(b.host(guestPath), mode)
}

func (b *Backend) MkdirAll(guestPath string, perm os.FileMode) error {
	if perm == 0 {
		perm = 0o755
	}
	return b.rt.MkdirAll(b.host(guestPath), perm)
}

func (b *Backend) ReadDir(guestPath string) ([]types.GuestDirEntry, error) {
	return b.rt.ReadDir(b.host(guestPath))
}

func (b *Backend) StatFS(guestPath string) (freeBytes, freeInodes int64, err error) {
	return b.rt.StatFS(b.host(guestPath))
}

// Upload copies a host file or directory tree into the guest at guestPath.
func (b *Backend) Upload(hostPath, guestPath string) error {
	info, err := os.Stat(hostPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		data, err := os.ReadFile(hostPath)
		if err != nil {
			return err
		}
		return b.rt.WriteFile(b.host(guestPath), data, info.Mode().Perm())
	}
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
			return b.MkdirAll(gp, fi.Mode().Perm())
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return b.rt.WriteFile(b.host(gp), data, fi.Mode().Perm())
	})
}

// Download copies a guest file to a host path.
func (b *Backend) Download(guestPath, hostPath string) error {
	data, err := b.rt.ReadFile(b.host(guestPath))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(hostPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(hostPath, data, 0o644)
}

// Sync flushes runtime writes to disk via sync(8).
func (b *Backend) Sync() error {
	_, err := b.run("sync")
	return err
}
