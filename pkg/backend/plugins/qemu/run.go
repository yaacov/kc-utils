//go:build unix

package qemu

import (
	"fmt"
	"log/slog"
	"path"
	"strings"
)

// RunCommand executes cmd inside the guest via chroot in the appliance. The
// guest's /proc, /sys, and /dev are bound before the command runs and unbound
// afterwards. Foreign-ISA guest ELFs are run by appliance binfmt/qemu-user;
// kernel/arch/OS still come from the guest tree (inspect, explicit dracut kver).
//
// guestRoot is the host mount root; the guest is chrooted at applianceMountRoot
// inside the appliance. cmd runs the guest's own binaries.
//
// RunCommand is not safe for concurrent use: it mounts shared virtual
// filesystems under applianceMountRoot for the duration of the command and
// unmounts them afterwards, so overlapping calls would tear down each other's
// mounts. The conversion pipeline invokes it sequentially.
func (b *Backend) RunCommand(_ string, cmd []string) ([]byte, error) {
	if err := b.mountVirtualFS(); err != nil {
		b.unmountVirtualFS()
		return nil, err
	}
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

// devFdLinks are created on the appliance /dev (the bind source) so a chroot
// without udev still has bash process substitution (/dev/fd/N). dracut's
// module-setup.sh uses that; missing links show up as /dev/fd/63 ENOENT.
var devFdLinks = [][]string{
	{"ln", "-sfn", "/proc/self/fd", "/dev/fd"},
	{"ln", "-sfn", "fd/0", "/dev/stdin"},
	{"ln", "-sfn", "fd/1", "/dev/stdout"},
	{"ln", "-sfn", "fd/2", "/dev/stderr"},
}

func (b *Backend) mountVirtualFS() error {
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
	return b.ensureDevFds()
}

func (b *Backend) ensureDevFds() error {
	for _, args := range devFdLinks {
		if _, err := b.session.client.run(args...); err != nil {
			if b.devFdLinkOK(args) {
				continue
			}
			return fmt.Errorf("ensure %s: %w", args[len(args)-1], err)
		}
	}
	return nil
}

func (b *Backend) devFdLinkOK(args []string) bool {
	if len(args) < 4 {
		return false
	}
	target, link := args[2], args[3]
	out, err := b.session.client.run("readlink", link)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == target
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
