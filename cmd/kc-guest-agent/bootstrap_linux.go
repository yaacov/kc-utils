//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/yaacov/kc-utils/pkg/qemuagent/server"
)

// defaultPort is the guest device file backed by the host unix socket via a
// QEMU virtio-serial port (see the qemu backend's launch args).
const defaultPort = "/dev/virtio-ports/org.kc-utils.agent"

type mountpoint struct {
	source, target, fstype string
	flags                  uintptr
}

// coreMounts are the pseudo-filesystems guest tools (blkid, lvm, mount, chroot)
// expect. Failures are non-fatal: a device may already be mounted by the kernel.
var coreMounts = []mountpoint{
	{"proc", "/proc", "proc", 0},
	{"sysfs", "/sys", "sysfs", 0},
	{"devtmpfs", "/dev", "devtmpfs", 0},
	{"tmpfs", "/run", "tmpfs", 0},
	{"tmpfs", "/tmp", "tmpfs", 0},
}

// virtioModules are loaded (best-effort) during init so the agent can reach the
// virtio-serial control port and the attached virtio-blk disks. Fedora ships
// these as modules; loading them here avoids depending on dracut/udev.
var virtioModules = []string{
	"virtio",
	"virtio_pci",
	"virtio_console", // /dev/virtio-ports/<name>
	"virtio_blk",     // /dev/vd*
	"virtio_scsi",
	"virtio_net",
}

// run serves the agent, optionally performing minimal PID-1 init first. It
// accepts sequential connections: the host reconnects once per pipeline stage,
// so on a clean disconnect the agent reopens the port and keeps serving.
// initPath is the PATH the agent runs guest tooling with. PID 1 inherits no
// environment, so without this exec.LookPath cannot find mkdir/mount/lvm/etc.
const initPath = "/usr/sbin:/usr/bin:/sbin:/bin"

func run(port string, asInit bool) error {
	if asInit {
		if os.Getenv("PATH") == "" {
			_ = os.Setenv("PATH", initPath)
		}
		mountCoreFilesystems()
		loadKernelModules()
	}
	for {
		err := serveOnce(port)
		if err != nil {
			fmt.Fprintln(os.Stderr, "kc-guest-agent: serve:", err)
		}
		if !asInit {
			// A one-shot (--port) run surfaces the serve error to the caller.
			return err
		}
		// PID 1 must never exit; wait briefly then re-open the port.
		time.Sleep(200 * time.Millisecond)
	}
}

func serveOnce(port string) error {
	dev, err := resolvePort(port)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(dev, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open %s: %w", dev, err)
	}
	defer f.Close()
	return server.Serve(f)
}

// resolvePort returns the device file for the agent's virtio-serial port. The
// named node /dev/virtio-ports/<name> is created by a udev rule we don't run in
// this minimal initramfs, so when it is absent we fall back to sysfs: match the
// port name against /sys/class/virtio-ports/*/name and open the raw
// /dev/vportNpM node that devtmpfs created for it.
func resolvePort(port string) (string, error) {
	if _, err := os.Stat(port); err == nil {
		return port, nil
	}
	want := filepath.Base(port)
	nameFiles, _ := filepath.Glob("/sys/class/virtio-ports/*/name")
	for _, nameFile := range nameFiles {
		data, err := os.ReadFile(nameFile)
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(data)) == want {
			return "/dev/" + filepath.Base(filepath.Dir(nameFile)), nil
		}
	}
	return "", fmt.Errorf("virtio-serial port %q not found (no /dev/%s and no /sys/class/virtio-ports match)", want, want)
}

func mountCoreFilesystems() {
	for _, m := range coreMounts {
		if err := os.MkdirAll(m.target, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "kc-guest-agent: mkdir %s: %v\n", m.target, err)
			continue
		}
		if err := syscall.Mount(m.source, m.target, m.fstype, m.flags, ""); err != nil {
			fmt.Fprintf(os.Stderr, "kc-guest-agent: mount %s: %v\n", m.target, err)
		}
	}
}

// loadKernelModules best-effort modprobes the virtio drivers. Errors are
// non-fatal: a module may be built into the kernel or already loaded.
func loadKernelModules() {
	for _, mod := range virtioModules {
		cmd := exec.Command("modprobe", mod)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "kc-guest-agent: modprobe %s: %v\n", mod, err)
		}
	}
}
