//go:build linux

package guest

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

// Guest is the high-level handle used by prepare/convert/finalize pipelines.
type Guest struct {
	rootPath string
	mode     Mode
	backend  Backend
}

// Open sets up disk access for the given mode and runs backend Setup.
func Open(disks []types.DiskSpec, mountRoot string, mode Mode) (*Guest, error) {
	if err := os.MkdirAll(mountRoot, 0o755); err != nil {
		return nil, fmt.Errorf("creating mount root %s: %w", mountRoot, err)
	}

	f, err := LookupFactory(mode.String())
	if err != nil {
		return nil, err
	}
	b, err := f.Open(disks, mountRoot)
	if err != nil {
		return nil, fmt.Errorf("backend setup: %w", err)
	}

	return &Guest{
		rootPath: mountRoot,
		mode:     mode,
		backend:  b,
	}, nil
}

// AttachFromPrepare sets up a guest handle from prepare output data.
// It derives the mode, orders disks, converts specs, attaches, and sets the
// global active handle. Callers must defer ClearActive().
func AttachFromPrepare(disks []types.DiskInfo, rootDevice, mountRoot string, backend string) (*Guest, error) {
	mode, err := ParseMode(backend)
	if err != nil {
		return nil, err
	}
	ordered := WithRootMount(disks, rootDevice)
	specs := types.DiskSpecsFrom(ordered)
	g, err := AttachMounted(specs, mountRoot, mode, ordered)
	if err != nil {
		return nil, fmt.Errorf("attach guest: %w", err)
	}
	SetActive(g)
	return g, nil
}

// AttachMounted returns a handle for a guest already prepared (mounted in
// direct mode, or with mount specs recorded for guestfs). Convert and finalize
// use this after prepare Release.
func AttachMounted(disks []types.DiskSpec, mountRoot string, mode Mode, diskInfos []types.DiskInfo) (*Guest, error) {
	f, err := LookupFactory(mode.String())
	if err != nil {
		return nil, err
	}
	b, err := f.Attach(disks, mountRoot, diskInfos)
	if err != nil {
		return nil, fmt.Errorf("attach backend: %w", err)
	}
	return &Guest{rootPath: mountRoot, mode: mode, backend: b}, nil
}

func (g *Guest) Mode() Mode       { return g.mode }
func (g *Guest) Backend() Backend { return g.backend }
func (g *Guest) RootPath() string { return g.rootPath }

// HostPath joins guestPath onto the mount root. In direct mode this is a live
// mount path. In guestfs mode it is only a path key for File* helpers and
// Checkout; it is not a populated host tree.
func (g *Guest) HostPath(guestPath string) string {
	return filepath.Join(g.rootPath, stringsTrimGuest(guestPath))
}

func stringsTrimGuest(guestPath string) string {
	p := filepath.FromSlash(normalizeGuestPath(guestPath))
	return stringsTrimPrefixSlash(p)
}

func stringsTrimPrefixSlash(p string) string {
	for len(p) > 0 && (p[0] == '/' || p[0] == filepath.Separator) {
		p = p[1:]
	}
	return p
}

func (g *Guest) DiskInfos() []types.DiskInfo { return g.backend.DiskInfos() }
func (g *Guest) LVPaths() []string           { return g.backend.LVPaths() }
func (g *Guest) DiskPaths() []string         { return g.backend.DiskPaths() }

func (g *Guest) ReadFile(guestPath string) ([]byte, error) {
	return g.backend.ReadFile(normalizeGuestPath(guestPath))
}

func (g *Guest) WriteFile(guestPath string, data []byte, perm os.FileMode) error {
	return g.backend.WriteFile(normalizeGuestPath(guestPath), data, perm)
}

func (g *Guest) Exists(guestPath string) bool {
	return g.backend.Exists(normalizeGuestPath(guestPath))
}

func (g *Guest) IsDir(guestPath string) bool {
	return g.backend.IsDir(normalizeGuestPath(guestPath))
}

func (g *Guest) Glob(pattern string) ([]string, error) {
	return g.backend.Glob(normalizeGuestPath(pattern))
}

func (g *Guest) Remove(guestPath string) error {
	return g.backend.Remove(normalizeGuestPath(guestPath))
}

func (g *Guest) RemoveAll(guestPath string) error {
	return g.backend.RemoveAll(normalizeGuestPath(guestPath))
}

func (g *Guest) Rename(oldPath, newPath string) error {
	return g.backend.Rename(normalizeGuestPath(oldPath), normalizeGuestPath(newPath))
}

func (g *Guest) Symlink(target, link string) error {
	return g.backend.Symlink(target, normalizeGuestPath(link))
}

func (g *Guest) Readlink(guestPath string) (string, error) {
	return g.backend.Readlink(normalizeGuestPath(guestPath))
}

func (g *Guest) Chmod(guestPath string, mode os.FileMode) error {
	return g.backend.Chmod(normalizeGuestPath(guestPath), mode)
}

func (g *Guest) Mkdir(guestPath string, perm os.FileMode) error {
	return g.backend.MkdirAll(normalizeGuestPath(guestPath), perm)
}

func (g *Guest) ReadDir(guestPath string) ([]DirEntry, error) {
	return g.backend.ReadDir(normalizeGuestPath(guestPath))
}

func (g *Guest) Upload(hostPath, guestPath string) error {
	return g.backend.Upload(hostPath, normalizeGuestPath(guestPath))
}

func (g *Guest) Download(guestPath, hostPath string) error {
	return g.backend.Download(normalizeGuestPath(guestPath), hostPath)
}

// StatFS returns free bytes and free inodes for the filesystem containing guestPath.
func (g *Guest) StatFS(guestPath string) (freeBytes, freeInodes int64, err error) {
	return g.backend.StatFS(normalizeGuestPath(guestPath))
}

// MountPartition mounts a single partition at the given guest mount point.
func (g *Guest) MountPartition(device, guestMountPoint string, readOnly bool) error {
	guestMountPoint = normalizeGuestPath(guestMountPoint)
	hostMountPoint := g.HostPath(guestMountPoint)
	ft, err := g.backend.FSType(device)
	if err != nil {
		return fmt.Errorf("detect fstype %s: %w", device, err)
	}
	return g.backend.Mount(device, hostMountPoint, ft, readOnly)
}

func (g *Guest) UnmountAll() error { return g.backend.UnmountAll() }

func (g *Guest) FSType(device string) (string, error) {
	return g.backend.FSType(device)
}

func (g *Guest) BlkidAttr(device, attr string) (string, error) {
	return g.backend.BlkidAttr(device, attr)
}

func (g *Guest) FSCheck(device, fstype string) error {
	return g.backend.FSCheck(device, fstype)
}

func (g *Guest) FSTrim(mountpoint string) error {
	return g.backend.FSTrim(mountpoint)
}

func (g *Guest) Decrypt(device, keyFile, mapperName string) (string, error) {
	return g.backend.Decrypt(device, keyFile, mapperName)
}

func (g *Guest) UnlockClevis(device, mapperName string) (string, error) {
	return g.backend.UnlockClevis(device, mapperName)
}

func (g *Guest) CloseCrypt(mapperName string) error {
	return g.backend.CloseCrypt(mapperName)
}

// RescanBlock refreshes LVM discovery after LUKS unlock so newly visible
// volumes appear in LVPaths.
func (g *Guest) RescanBlock() error {
	return g.backend.RescanBlock()
}

func (g *Guest) RunCommand(guestRoot string, cmd []string) ([]byte, error) {
	return g.backend.RunCommand(normalizeGuestPath(guestRoot), cmd)
}

func (g *Guest) DeviceRead(device string, offset int64, size int) ([]byte, error) {
	return g.backend.DeviceRead(device, offset, size)
}

func (g *Guest) DeviceWrite(device string, offset int64, data []byte) error {
	return g.backend.DeviceWrite(device, offset, data)
}

// ProbeMount downloads OS-root markers to a host temp dir for inspection.
// The path is outside RootPath so File* helpers do not treat it as a guest path.
func (g *Guest) ProbeMount(device, fstype string) (hostPath string, err error) {
	hostPath = filepath.Join(os.TempDir(), "kc-scan-"+sanitizeDevice(device))
	if err := os.MkdirAll(hostPath, 0o755); err != nil {
		return "", err
	}
	if err := g.backend.ProbeMount(device, fstype, hostPath); err != nil {
		return "", err
	}
	return hostPath, nil
}

func (g *Guest) ProbeUnmount(hostPath string) error {
	return g.backend.ProbeUnmount(hostPath)
}

// Sync is a no-op; guestfs file writes go through the appliance live.
func (g *Guest) Sync() error {
	return g.backend.Sync()
}

// Release frees process-local state without tearing down mounts for convert.
func (g *Guest) Release() error {
	return g.backend.Release()
}

// UnmountFilesystems unmounts guest filesystems but keeps block/LUKS/LVM
// devices open (finalize path, before post-fsck).
func (g *Guest) UnmountFilesystems() error {
	return g.backend.UnmountFilesystems()
}

// ReleaseDevices closes LUKS, deactivates LVM, and detaches loop devices
// (finalize path, after post-fsck).
func (g *Guest) ReleaseDevices() error {
	return g.backend.ReleaseDevices()
}

// Teardown fully cleans up mounts and devices (finalize).
func (g *Guest) Teardown() error {
	return g.backend.Teardown()
}

// TeardownDiscard reclaims host resources without writing guest edits
// back to disk images. Used for failure cleanup.
func (g *Guest) TeardownDiscard() error {
	return g.backend.TeardownDiscard()
}

func sanitizeDevice(device string) string {
	out := make([]byte, 0, len(device))
	for _, c := range device {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			out = append(out, byte(c))
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}
