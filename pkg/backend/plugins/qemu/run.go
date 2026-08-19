//go:build unix

package qemu

import (
	"fmt"
	"log/slog"
	"path"
)

// RunCommand executes cmd inside the guest via chroot in the appliance. The
// guest's /proc, /sys, and /dev are provided before the command runs and torn
// down afterwards, matching virt-v2v behaviour (and the guestfs backend).
//
// guestRoot is the host mount root; the guest is chrooted at applianceMountRoot
// inside the appliance. cmd runs the guest's own (guest-arch) binaries.
//
// RunCommand is not safe for concurrent use: it mounts shared virtual
// filesystems under applianceMountRoot for the duration of the command and
// unmounts them afterwards, so overlapping calls would tear down each other's
// mounts. The conversion pipeline invokes it sequentially.
func (b *Backend) RunCommand(_ string, cmd []string) ([]byte, error) {
	b.mountVirtualFS()
	defer b.unmountVirtualFS()

	argv := append([]string{"chroot", applianceMountRoot}, cmd...)
	res, err := b.session.client.exec(argv, nil, nil)
	if err != nil {
		return nil, err
	}
	out := append(append([]byte(nil), res.Stdout...), res.Stderr...)
	if res.ExitCode != 0 {
		return out, fmt.Errorf("guest command %v: exit %d", cmd, res.ExitCode)
	}
	return out, nil
}

// virtualFS lists the guest virtual filesystems mounted for chrooted commands.
var virtualFS = []struct {
	source string
	fstype string
	target string // relative to applianceMountRoot
	bind   bool
}{
	{source: "proc", fstype: "proc", target: "proc"},
	{source: "sysfs", fstype: "sysfs", target: "sys"},
	{source: "/dev", target: "dev", bind: true},
}

func (b *Backend) mountVirtualFS() {
	for _, vfs := range virtualFS {
		target := path.Join(applianceMountRoot, vfs.target)
		if _, err := b.session.client.run("mkdir", "-p", target); err != nil {
			slog.Warn("mkdir virtual fs mountpoint failed", "target", target, "error", err)
			continue
		}
		var argv []string
		if vfs.bind {
			argv = []string{"mount", "--bind", vfs.source, target}
		} else {
			argv = []string{"mount", "-t", vfs.fstype, vfs.source, target}
		}
		if _, err := b.session.client.run(argv...); err != nil {
			slog.Warn("mount virtual fs failed", "target", target, "error", err)
		}
	}
}

func (b *Backend) unmountVirtualFS() {
	// Unmount in reverse so /dev (bind) is released before its parent.
	for i := len(virtualFS) - 1; i >= 0; i-- {
		target := path.Join(applianceMountRoot, virtualFS[i].target)
		if _, err := b.session.client.run("umount", target); err != nil {
			slog.Debug("unmount virtual fs failed", "target", target, "error", err)
		}
	}
}
