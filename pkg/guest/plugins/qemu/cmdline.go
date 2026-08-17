//go:build unix

package qemu

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/yaacov/kc-utils/pkg/agent/protocol"
	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest/backend"
)

// LaunchConfig describes a QEMU appliance process.
type LaunchConfig struct {
	Arch     string
	Accel    string
	Kernel   string
	Initrd   string
	Assets   string
	Socket   string
	Disks    []types.DiskSpec
	MemoryMB int
	SMP      int
	Network  bool
	Machine  string
	QEMUBin  string
}

func defaultArch() string {
	switch runtime.GOARCH {
	case "arm64", "aarch64":
		return "arm64"
	default:
		return "amd64"
	}
}

func defaultAccel() string {
	switch runtime.GOOS {
	case "darwin":
		return "hvf"
	case "linux":
		if _, err := os.Stat("/dev/kvm"); err == nil {
			return "kvm"
		}
	}
	return "tcg"
}

func qemuBinary(arch string) string {
	if arch == "arm64" {
		return "qemu-system-aarch64"
	}
	return "qemu-system-x86_64"
}

func machineFor(arch string) string {
	if arch == "arm64" {
		return "virt"
	}
	return "q35"
}

func consoleFor(arch string) string {
	if arch == "arm64" {
		return "ttyAMA0"
	}
	return "ttyS0"
}

func envPositiveInt(def int, names ...string) int {
	for _, name := range names {
		v := strings.TrimSpace(os.Getenv(name))
		if v == "" {
			continue
		}
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			return n
		}
	}
	return def
}

func resolveMemSize() int {
	return envPositiveInt(2048, "V2V_memSize", "LIBGUESTFS_MEMSIZE")
}

func resolveSMP() int {
	smp := runtime.NumCPU()
	if smp > 8 {
		smp = 8
	}
	if smp < 1 {
		smp = 1
	}
	return envPositiveInt(smp, "V2V_smp", "LIBGUESTFS_SMP")
}

func applianceDir() string {
	if d := strings.TrimSpace(os.Getenv(backend.EnvApplianceDir)); d != "" {
		return d
	}
	exe, err := os.Executable()
	if err == nil {
		return filepath.Join(filepath.Dir(exe), "appliance", defaultArch())
	}
	return filepath.Join("build", "appliance", "out", defaultArch())
}

func resolveArtifacts() (kernel, initrd, assets string, err error) {
	dir := applianceDir()
	kernel = filepath.Join(dir, "vmlinuz")
	initrd = filepath.Join(dir, "initramfs.img")
	for _, p := range []string{kernel, initrd} {
		if _, statErr := os.Stat(p); statErr != nil {
			return "", "", "", fmt.Errorf("appliance artifact %s: %w (set %s)", p, statErr, backend.EnvApplianceDir)
		}
	}
	assets = filepath.Join(dir, "assets.squashfs")
	if _, statErr := os.Stat(assets); statErr != nil {
		assets = ""
	}
	return kernel, initrd, assets, nil
}

func guestfsNetworkEnabled() bool {
	v := strings.TrimSpace(os.Getenv(backend.EnvGuestfsNetwork))
	return v == "1" || strings.EqualFold(v, "true")
}

// BuildQEMUArgs returns the qemu-system binary and argv for cfg.
func BuildQEMUArgs(cfg *LaunchConfig) (string, []string, error) {
	if cfg == nil {
		return "", nil, fmt.Errorf("launch config is required")
	}
	if cfg.Arch == "" {
		cfg.Arch = defaultArch()
	}
	if cfg.Accel == "" {
		cfg.Accel = defaultAccel()
	}
	if cfg.QEMUBin == "" {
		cfg.QEMUBin = qemuBinary(cfg.Arch)
	}
	if cfg.Machine == "" {
		cfg.Machine = machineFor(cfg.Arch)
	}
	if cfg.MemoryMB <= 0 {
		cfg.MemoryMB = 2048
	}
	if cfg.SMP <= 0 {
		cfg.SMP = 1
	}
	if cfg.Kernel == "" || cfg.Initrd == "" || cfg.Socket == "" {
		return "", nil, fmt.Errorf("kernel, initrd, and socket are required")
	}

	cpu := "max"
	if cfg.Accel == "hvf" || cfg.Accel == "kvm" {
		cpu = "host"
	}

	args := []string{
		"-nodefaults",
		"-nographic",
		"-no-reboot",
		"-machine", cfg.Machine,
		"-cpu", cpu,
		"-accel", cfg.Accel,
		"-m", strconv.Itoa(cfg.MemoryMB),
		"-smp", strconv.Itoa(cfg.SMP),
		"-kernel", cfg.Kernel,
		"-initrd", cfg.Initrd,
		"-append", "rdinit=/kc-agent console=" + consoleFor(cfg.Arch) + " quiet",
		"-chardev", "socket,id=agent,path=" + cfg.Socket + ",server=on,wait=off",
	}
	if cfg.Arch == "arm64" {
		args = append(args,
			"-device", "virtio-serial-device",
			"-device", "virtserialport,chardev=agent,name="+protocol.PortName,
		)
	} else {
		args = append(args,
			"-device", "virtio-serial-pci",
			"-device", "virtserialport,chardev=agent,name="+protocol.PortName,
		)
	}

	blkDevice := "virtio-blk-pci"
	if cfg.Arch == "arm64" {
		blkDevice = "virtio-blk-device"
	}

	for i, d := range cfg.Disks {
		id := fmt.Sprintf("hd%d", i)
		format := d.Format
		if format == "" {
			format = "raw"
		}
		args = append(args,
			"-drive", fmt.Sprintf("file=%s,if=none,id=%s,format=%s,cache=none", d.Path, id, format),
			"-device", fmt.Sprintf("%s,drive=%s,serial=kc-disk-%d", blkDevice, id, i),
		)
	}
	if cfg.Assets != "" {
		args = append(args,
			"-drive", fmt.Sprintf("file=%s,if=none,id=assets,format=raw,readonly=on", cfg.Assets),
			"-device", fmt.Sprintf("%s,drive=assets,serial=kc-assets", blkDevice),
		)
	}
	if cfg.Network {
		netDev := "virtio-net-pci"
		if cfg.Arch == "arm64" {
			netDev = "virtio-net-device"
		}
		args = append(args,
			"-netdev", "user,id=net0",
			"-device", netDev+",netdev=net0",
		)
	}
	return cfg.QEMUBin, args, nil
}
