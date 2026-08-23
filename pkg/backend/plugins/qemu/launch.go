//go:build unix

package qemu

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Appliance-side conventions shared between launch and the running backend.
const (
	// agentPortName is the virtio-serial port the in-guest agent listens on;
	// it is bridged to a host unix socket by the -chardev/-device pair below.
	agentPortName = "org.kc-utils.agent"

	// debugPortName is a second virtio-serial port used as an interactive
	// channel (PID 1 binds /bin/bash to it). Bridged to debug.sock next to
	// the agent socket.
	debugPortName = "org.kc-utils.debug"

	// debugSockName is the unix-socket filename sitting next to agent.sock.
	debugSockName = "debug.sock"

	// applianceMountRoot is where guest filesystems are mounted inside the
	// appliance. Host mount points are rebased under it (see mount.go / fs.go).
	applianceMountRoot = "/mnt/guest"

	// defaultApplianceDir is where the appliance kernel + initramfs are shipped
	// in the container image; overridable via EnvApplianceDir.
	defaultApplianceDir = "/usr/lib/kc-utils/appliance"
)

// Environment overrides for locating and sizing the appliance.
const (
	EnvApplianceDir  = "KC_APPLIANCE_DIR"
	EnvApplianceArch = "KC_APPLIANCE_ARCH"
	EnvQEMUBinary    = "KC_QEMU_BINARY"
)

// driveSpec is one guest disk attached to the appliance as a virtio-blk drive.
type driveSpec struct {
	Path   string
	Format string // "qcow2", "raw"; empty lets qemuArgs fall back to "raw"
}

// launchConfig is the fully-resolved input to qemuArgs.
type launchConfig struct {
	Binary          string
	Machine         string
	CPU             string
	Accel           string // "hvf", "kvm", or "tcg"
	MemMiB          int
	SMP             int
	Kernel          string
	Initrd          string
	Cmdline         string
	SocketPath      string
	DebugSocketPath string // sibling of SocketPath; interactive debug channel
	Drives          []driveSpec
	Network         bool // user-mode networking (Clevis/NBDE)
}

// debugSocketPath returns the debug-channel unix socket that sits next to the
// agent socket in the same private directory.
func debugSocketPath(agentSock string) string {
	return filepath.Join(filepath.Dir(agentSock), debugSockName)
}

// applianceArch returns the appliance architecture (Go GOARCH naming),
// defaulting to the host arch so hardware acceleration applies.
func applianceArch() string {
	if a := os.Getenv(EnvApplianceArch); a != "" {
		return a
	}
	return runtime.GOARCH
}

// qemuBinary returns the qemu-system binary for the given Go arch.
func qemuBinary(arch string) string {
	if b := os.Getenv(EnvQEMUBinary); b != "" {
		return b
	}
	switch arch {
	case "arm64":
		return "qemu-system-aarch64"
	case "amd64":
		return "qemu-system-x86_64"
	default:
		return "qemu-system-" + arch
	}
}

// accelFor picks the accelerator: KVM on Linux, HVF on macOS when hardware
// virtualization is available, else TCG emulation. accelAvailable is the
// result of the host probe (backend.Requirements.Accel / Probes.HasAccel).
func accelFor(goos string, accelAvailable bool) string {
	if !accelAvailable {
		return "tcg"
	}
	switch goos {
	case "linux":
		return "kvm"
	case "darwin":
		return "hvf"
	default:
		return "tcg"
	}
}

// machineFor returns the qemu machine type for a Go arch.
func machineFor(arch string) string {
	switch arch {
	case "arm64":
		return "virt"
	default:
		return "q35"
	}
}

// cpuFor returns the -cpu model. With hardware acceleration "host" passes the
// physical CPU through; under TCG a concrete model is required (host is invalid).
func cpuFor(arch, accel string) string {
	if accel != "tcg" {
		return "host"
	}
	switch arch {
	case "arm64":
		return "cortex-a72"
	default:
		return "max"
	}
}

// consoleFor returns the kernel console device for a Go arch.
func consoleFor(arch string) string {
	switch arch {
	case "arm64":
		return "ttyAMA0"
	default:
		return "ttyS0"
	}
}

// kernelCmdline builds the appliance kernel command line. The initramfs runs
// our agent as /init; a serial console aids boot diagnostics; panic=1 makes a
// failed boot exit promptly instead of hanging.
func kernelCmdline(arch string) string {
	return "console=" + consoleFor(arch) + " panic=1 loglevel=4"
}

// appliancePaths locates the kernel (vmlinuz) and initramfs for arch under the
// appliance directory: <dir>/<arch>/{vmlinuz,initramfs.img}.
func appliancePaths(arch string) (kernel, initrd string, err error) {
	dir := os.Getenv(EnvApplianceDir)
	if dir == "" {
		dir = defaultApplianceDir
	}
	archDir := filepath.Join(dir, arch)
	kernel = filepath.Join(archDir, "vmlinuz")
	initrd = filepath.Join(archDir, "initramfs.img")
	for _, p := range []string{kernel, initrd} {
		if _, statErr := os.Stat(p); statErr != nil {
			return "", "", fmt.Errorf("appliance image not found: %s (set %s)", p, EnvApplianceDir)
		}
	}
	return kernel, initrd, nil
}

// escapeQEMUValue escapes a value embedded in a comma-separated QEMU option
// string. QEMU treats a comma as an option separator, so a literal comma in a
// value (e.g. a file path) must be doubled.
func escapeQEMUValue(s string) string {
	return strings.ReplaceAll(s, ",", ",,")
}

// driveFormat returns the qemu format string for a drive, defaulting to raw.
func driveFormat(d driveSpec) string {
	if d.Format == "" {
		return "raw"
	}
	return d.Format
}

// qemuArgs builds the full qemu-system argument list. It is pure so it can be
// unit-tested without launching anything.
func qemuArgs(c *launchConfig) []string {
	args := []string{
		"-machine", c.Machine + ",accel=" + c.Accel,
		"-cpu", c.CPU,
		"-m", strconv.Itoa(c.MemMiB),
		"-smp", strconv.Itoa(c.SMP),
		"-kernel", c.Kernel,
		"-initrd", c.Initrd,
		"-append", c.Cmdline,
		"-nodefaults",
		"-no-reboot",
		"-display", "none",
		"-serial", "stdio",
	}

	// Host unix socket <-> guest virtio-serial port carrying the agent protocol.
	// QEMU owns (server=on) and creates the socket; the host connects as client.
	args = append(args,
		"-chardev", "socket,id=kcagent,path="+escapeQEMUValue(c.SocketPath)+",server=on,wait=off",
		"-device", "virtio-serial",
		"-device", "virtserialport,chardev=kcagent,name="+agentPortName,
	)

	// Interactive debug channel: same virtio-serial controller, second port.
	// The in-guest agent binds an app (bash) to it; the host attaches with socat.
	debugSock := c.DebugSocketPath
	if debugSock == "" && c.SocketPath != "" {
		debugSock = debugSocketPath(c.SocketPath)
	}
	if debugSock != "" {
		args = append(args,
			"-chardev", "socket,id=kcdebug,path="+escapeQEMUValue(debugSock)+",server=on,wait=off",
			"-device", "virtserialport,chardev=kcdebug,name="+debugPortName,
		)
	}

	// Guest disks as virtio-blk drives, attached in order -> /dev/vd{a,b,...}.
	for i, d := range c.Drives {
		id := "disk" + strconv.Itoa(i)
		args = append(args,
			"-drive", "if=none,id="+id+",file="+escapeQEMUValue(d.Path)+",format="+driveFormat(d)+",cache=writeback",
			"-device", "virtio-blk-pci,drive="+id,
		)
	}

	if c.Network {
		// User-mode networking; only enabled for Clevis/NBDE Tang access.
		args = append(args,
			"-netdev", "user,id=net0",
			"-device", "virtio-net-pci,netdev=net0",
		)
	}

	return args
}
