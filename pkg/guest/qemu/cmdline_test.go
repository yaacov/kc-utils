//go:build unix

package qemu

import (
	"runtime"
	"strings"
	"testing"
)

func TestBuildQEMUArgsAmd64(t *testing.T) {
	bin, args, err := BuildQEMUArgs(&LaunchConfig{
		Arch:     "amd64",
		Accel:    "tcg",
		Kernel:   "/k/vmlinuz",
		Initrd:   "/k/initramfs.img",
		Socket:   "/tmp/agent.sock",
		MemoryMB: 1024,
		SMP:      2,
		Disks:    nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	if bin != "qemu-system-x86_64" {
		t.Fatalf("bin %s", bin)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"-machine q35",
		"-accel tcg",
		"-kernel /k/vmlinuz",
		"rdinit=/kc-agent",
		"org.kc-utils.agent",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %s", want, joined)
		}
	}
	if strings.Contains(joined, "serial=kc-assets") {
		t.Fatalf("did not expect assets disk: %s", joined)
	}
}

func TestBuildQEMUArgsArm64(t *testing.T) {
	bin, args, err := BuildQEMUArgs(&LaunchConfig{
		Arch:   "arm64",
		Accel:  "hvf",
		Kernel: "/k/vmlinuz",
		Initrd: "/k/initramfs.img",
		Socket: "/tmp/agent.sock",
	})
	if err != nil {
		t.Fatal(err)
	}
	if bin != "qemu-system-aarch64" {
		t.Fatalf("bin %s", bin)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-machine virt") {
		t.Fatalf("want virt machine: %s", joined)
	}
	if !strings.Contains(joined, "-accel hvf") {
		t.Fatalf("want hvf: %s", joined)
	}
	if !strings.Contains(joined, "virtio-serial-device") {
		t.Fatalf("want virtio-serial-device: %s", joined)
	}
}

func TestBuildQEMUArgsRequiresPaths(t *testing.T) {
	if _, _, err := BuildQEMUArgs(&LaunchConfig{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestDefaultAccel(t *testing.T) {
	got := defaultAccel()
	switch runtime.GOOS {
	case "darwin":
		if got != "hvf" {
			t.Fatalf("darwin accel %s, want hvf", got)
		}
	case "linux":
		if got != "kvm" && got != "tcg" {
			t.Fatalf("linux accel %s", got)
		}
	}
}

func TestBuildQEMUArgsNetwork(t *testing.T) {
	_, args, err := BuildQEMUArgs(&LaunchConfig{
		Arch:    "amd64",
		Kernel:  "/k/vmlinuz",
		Initrd:  "/k/initramfs.img",
		Socket:  "/tmp/agent.sock",
		Network: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "user,id=net0") {
		t.Fatalf("want user netdev: %s", joined)
	}
}

func TestBuildQEMUArgsOptionalAssets(t *testing.T) {
	_, args, err := BuildQEMUArgs(&LaunchConfig{
		Arch:   "amd64",
		Kernel: "/k/vmlinuz",
		Initrd: "/k/initramfs.img",
		Assets: "/k/assets.squashfs",
		Socket: "/tmp/agent.sock",
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "serial=kc-assets") {
		t.Fatalf("want assets disk: %s", joined)
	}
}
