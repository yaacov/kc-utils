//go:build unix

package qemu

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestAccelFor(t *testing.T) {
	cases := []struct {
		goos      string
		available bool
		want      string
	}{
		{"linux", true, "kvm"},
		{"darwin", true, "hvf"},
		{"linux", false, "tcg"},
		{"darwin", false, "tcg"},
		{"windows", true, "tcg"},
	}
	for _, c := range cases {
		if got := accelFor(c.goos, c.available); got != c.want {
			t.Errorf("accelFor(%q, %v) = %q, want %q", c.goos, c.available, got, c.want)
		}
	}
}

func TestMachineFor(t *testing.T) {
	if got := machineFor("arm64"); got != "virt" {
		t.Errorf("machineFor(arm64) = %q, want virt", got)
	}
	if got := machineFor("amd64"); got != "q35" {
		t.Errorf("machineFor(amd64) = %q, want q35", got)
	}
}

func TestCPUFor(t *testing.T) {
	// Hardware acceleration passes the physical CPU through.
	if got := cpuFor("arm64", "hvf"); got != "host" {
		t.Errorf("cpuFor(arm64, hvf) = %q, want host", got)
	}
	if got := cpuFor("amd64", "kvm"); got != "host" {
		t.Errorf("cpuFor(amd64, kvm) = %q, want host", got)
	}
	// TCG needs a concrete model; "host" is invalid there.
	if got := cpuFor("arm64", "tcg"); got != "cortex-a72" {
		t.Errorf("cpuFor(arm64, tcg) = %q, want cortex-a72", got)
	}
	if got := cpuFor("amd64", "tcg"); got != "max" {
		t.Errorf("cpuFor(amd64, tcg) = %q, want max", got)
	}
}

func TestConsoleForAndCmdline(t *testing.T) {
	if got := consoleFor("arm64"); got != "ttyAMA0" {
		t.Errorf("consoleFor(arm64) = %q, want ttyAMA0", got)
	}
	if got := consoleFor("amd64"); got != "ttyS0" {
		t.Errorf("consoleFor(amd64) = %q, want ttyS0", got)
	}
	cmd := kernelCmdline("arm64")
	for _, want := range []string{"console=ttyAMA0", "panic=1"} {
		if !strings.Contains(cmd, want) {
			t.Errorf("kernelCmdline(arm64) = %q, missing %q", cmd, want)
		}
	}
}

func TestQEMUBinary(t *testing.T) {
	t.Setenv(EnvQEMUBinary, "")
	if got := qemuBinary("arm64"); got != "qemu-system-aarch64" {
		t.Errorf("qemuBinary(arm64) = %q", got)
	}
	if got := qemuBinary("amd64"); got != "qemu-system-x86_64" {
		t.Errorf("qemuBinary(amd64) = %q", got)
	}
	if got := qemuBinary("riscv64"); got != "qemu-system-riscv64" {
		t.Errorf("qemuBinary(riscv64) = %q", got)
	}
	// Explicit override wins.
	t.Setenv(EnvQEMUBinary, "/opt/qemu")
	if got := qemuBinary("arm64"); got != "/opt/qemu" {
		t.Errorf("qemuBinary override = %q, want /opt/qemu", got)
	}
}

func TestDriveFormat(t *testing.T) {
	if got := driveFormat(driveSpec{Path: "/d.img"}); got != "raw" {
		t.Errorf("driveFormat(empty) = %q, want raw", got)
	}
	if got := driveFormat(driveSpec{Path: "/d.qcow2", Format: "qcow2"}); got != "qcow2" {
		t.Errorf("driveFormat(qcow2) = %q, want qcow2", got)
	}
}

func TestAppliancePaths(t *testing.T) {
	dir := t.TempDir()
	archDir := filepath.Join(dir, "arm64")
	if err := os.MkdirAll(archDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvApplianceDir, dir)

	// Missing images -> error.
	if _, _, err := appliancePaths("arm64"); err == nil {
		t.Fatal("expected error for missing appliance images")
	}

	kernel := filepath.Join(archDir, "vmlinuz")
	initrd := filepath.Join(archDir, "initramfs.img")
	for _, p := range []string{kernel, initrd} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gotK, gotI, err := appliancePaths("arm64")
	if err != nil {
		t.Fatalf("appliancePaths: %v", err)
	}
	if gotK != kernel || gotI != initrd {
		t.Errorf("appliancePaths = (%q, %q), want (%q, %q)", gotK, gotI, kernel, initrd)
	}
}

// flagValue returns the argument following the first occurrence of flag.
func flagValue(args []string, flag string) (string, bool) {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag {
			return args[i+1], true
		}
	}
	return "", false
}

func TestQEMUArgsCore(t *testing.T) {
	cfg := launchConfig{
		Binary:     "qemu-system-aarch64",
		Machine:    "virt",
		CPU:        "host",
		Accel:      "hvf",
		MemMiB:     2048,
		SMP:        4,
		Kernel:     "/a/vmlinuz",
		Initrd:     "/a/initramfs.img",
		Cmdline:    "console=ttyAMA0 panic=1",
		SocketPath: "/tmp/agent.sock",
		Drives: []driveSpec{
			{Path: "/disk0.qcow2", Format: "qcow2"},
			{Path: "/disk1.img"},
		},
	}
	args := qemuArgs(&cfg)

	if v, _ := flagValue(args, "-machine"); v != "virt,accel=hvf" {
		t.Errorf("-machine = %q", v)
	}
	if v, _ := flagValue(args, "-cpu"); v != "host" {
		t.Errorf("-cpu = %q", v)
	}
	if v, _ := flagValue(args, "-m"); v != "2048" {
		t.Errorf("-m = %q", v)
	}
	if v, _ := flagValue(args, "-smp"); v != "4" {
		t.Errorf("-smp = %q", v)
	}
	if v, _ := flagValue(args, "-kernel"); v != "/a/vmlinuz" {
		t.Errorf("-kernel = %q", v)
	}
	if v, _ := flagValue(args, "-initrd"); v != "/a/initramfs.img" {
		t.Errorf("-initrd = %q", v)
	}
	if v, _ := flagValue(args, "-append"); v != "console=ttyAMA0 panic=1" {
		t.Errorf("-append = %q", v)
	}

	// The agent chardev must be a server socket that does not block on boot.
	chardev, ok := flagValue(args, "-chardev")
	if !ok || !strings.Contains(chardev, "path=/tmp/agent.sock") ||
		!strings.Contains(chardev, "server=on") || !strings.Contains(chardev, "wait=off") {
		t.Errorf("-chardev = %q", chardev)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "virtserialport,chardev=kcagent,name="+agentPortName) {
		t.Errorf("missing virtserialport for agent: %q", joined)
	}

	// Two drives -> two if=none drives with matching virtio-blk devices, formats preserved.
	if strings.Count(joined, "if=none,") != 2 {
		t.Errorf("expected 2 drives, args=%q", joined)
	}
	if !strings.Contains(joined, "file=/disk0.qcow2,format=qcow2") {
		t.Errorf("disk0 format not qcow2: %q", joined)
	}
	if !strings.Contains(joined, "file=/disk1.img,format=raw") {
		t.Errorf("disk1 format not raw (default): %q", joined)
	}
	if strings.Count(joined, "virtio-blk-pci,drive=disk") != 2 {
		t.Errorf("expected 2 virtio-blk devices: %q", joined)
	}

	// -no-reboot and no graphics are required for a headless one-shot appliance.
	for _, want := range []string{"-no-reboot", "-nodefaults"} {
		if !slices.Contains(args, want) {
			t.Errorf("args missing %q: %v", want, args)
		}
	}

	// Networking is off by default.
	if strings.Contains(joined, "-netdev") {
		t.Errorf("networking should be disabled by default: %q", joined)
	}
}

func TestQEMUArgsNetworkEnabled(t *testing.T) {
	args := qemuArgs(&launchConfig{Machine: "q35", CPU: "max", Accel: "tcg", Network: true})
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-netdev") || !strings.Contains(joined, "user,id=net0") {
		t.Errorf("expected user networking when enabled: %q", joined)
	}
	if !strings.Contains(joined, "virtio-net-pci,netdev=net0") {
		t.Errorf("expected virtio-net device: %q", joined)
	}
}
