//go:build linux

package guest

import (
	"os"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

// DirEntry is a directory entry from ReadDir (guest paths).
type DirEntry = types.GuestDirEntry

// Backend abstracts privileged disk/mount operations so direct (host syscall)
// and guestfs (libguestfs) implementations can be swapped by Mode.
//
// All guestfs CLI and privileged host tool invocations for guest disks must
// live in Backend implementations under this package.
type Backend interface {
	// Setup opens disk image(s) and discovers partitions and LVM volumes.
	Setup(disks []types.DiskSpec, mountRoot string) error

	DiskInfos() []types.DiskInfo
	LVPaths() []string
	DiskPaths() []string

	Mount(device, hostMountPoint, fstype string, readOnly bool) error
	UnmountAll() error

	// ProbeMount temporarily mounts device read-only for inspection and
	// returns the host path. Caller must call ProbeUnmount when done.
	ProbeMount(device, fstype, hostMountPoint string) error
	ProbeUnmount(hostMountPoint string) error

	FSType(device string) (string, error)
	BlkidAttr(device, attr string) (string, error)
	FSCheck(device, fstype string) error
	FSTrim(mountpoint string) error

	// Decrypt opens a LUKS mapping with a key file.
	Decrypt(device, keyFile, mapperName string) (mappedPath string, err error)
	// UnlockClevis unlocks a Clevis-bound LUKS volume (NBDE / Tang).
	// Guestfs mode requires appliance networking (see EnvGuestfsNetwork).
	UnlockClevis(device, mapperName string) (mappedPath string, err error)
	CloseCrypt(mapperName string) error

	// RescanBlock refreshes LVM (and related) discovery after LUKS unlock.
	RescanBlock() error

	// RunCommand executes cmd in the guest (chroot or guestfish command).
	RunCommand(guestRoot string, cmd []string) ([]byte, error)

	// DeviceRead/DeviceWrite access raw partition bytes (for NTFS boot-sector patch etc.).
	DeviceRead(device string, offset int64, size int) ([]byte, error)
	DeviceWrite(device string, offset int64, data []byte) error

	// Guest filesystem operations. guestPath is absolute inside the guest (/etc/fstab).
	ReadFile(guestPath string) ([]byte, error)
	WriteFile(guestPath string, data []byte, perm os.FileMode) error
	Exists(guestPath string) bool
	IsDir(guestPath string) bool
	Glob(pattern string) ([]string, error) // guest-absolute paths
	Remove(guestPath string) error
	RemoveAll(guestPath string) error
	Rename(oldPath, newPath string) error
	Symlink(target, link string) error
	Readlink(guestPath string) (string, error)
	Chmod(guestPath string, mode os.FileMode) error
	MkdirAll(guestPath string, perm os.FileMode) error
	ReadDir(guestPath string) ([]DirEntry, error)
	// Upload copies a host file or directory into the guest at guestPath.
	Upload(hostPath, guestPath string) error
	// Download copies a guest file or directory to a host path.
	Download(guestPath, hostPath string) error

	// StatFS returns free bytes and free inodes for the filesystem containing guestPath.
	StatFS(guestPath string) (freeBytes, freeInodes int64, err error)

	// Sync is a no-op; guestfs writes go through the appliance live.
	Sync() error

	// Release frees process-local state without tearing down mounts that
	// must survive for convert (prepare exit path).
	Release() error

	// UnmountFilesystems unmounts guest filesystems but keeps LUKS mappers,
	// LVM volumes, and loop devices open (for post-fsck in finalize).
	UnmountFilesystems() error

	// ReleaseDevices closes LUKS, deactivates LVM, and detaches loop devices.
	ReleaseDevices() error

	// Teardown fully unmounts and releases devices (finalize path).
	Teardown() error

	// TeardownDiscard reclaims host resources without writing guest edits
	// back to disk images (failure cleanup path).
	TeardownDiscard() error
}
