//go:build unix

package qemu

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

// TestE2EApplianceRoundTrip boots the real appliance under the host hypervisor
// (HVF/KVM/TCG) and exercises the full primitive round-trip: VM boot + agent
// handshake, exec, on-disk mkfs, mount, file I/O, discovery, statfs, teardown.
//
// It is the local inner-loop check from the plan's Verification section. It is
// skipped unless KC_QEMU_E2E=1 and the appliance images for the host arch are
// installed (build/kc-appliance/build.sh, then point KC_APPLIANCE_DIR at
// bin/appliance).
func TestE2EApplianceRoundTrip(t *testing.T) {
	if os.Getenv("KC_QEMU_E2E") != "1" {
		t.Skip("set KC_QEMU_E2E=1 to run the appliance end-to-end test")
	}
	if _, _, err := appliancePaths(applianceArch()); err != nil {
		t.Skipf("appliance images not found: %v (build them and set %s)", err, EnvApplianceDir)
	}

	// A blank raw disk the appliance will partition-less format as whole-disk ext4.
	diskDir := t.TempDir()
	diskPath := filepath.Join(diskDir, "disk0.img")
	if err := os.Truncate(diskPath, 64<<20); err != nil { // create then size to 64 MiB
		f, cerr := os.Create(diskPath)
		if cerr != nil {
			t.Fatalf("create disk: %v", cerr)
		}
		if terr := f.Truncate(64 << 20); terr != nil {
			t.Fatalf("size disk: %v", terr)
		}
		_ = f.Close()
	}

	b := New()
	mountRoot := t.TempDir()
	disks := []types.DiskSpec{{Path: diskPath, Format: "raw"}}
	if err := b.Setup(disks, mountRoot); err != nil {
		t.Fatalf("Setup (boot + handshake + discovery): %v", err)
	}
	t.Cleanup(func() {
		if err := b.Teardown(); err != nil {
			t.Errorf("Teardown: %v", err)
		}
	})

	// Sanity: the agent can exec a guest binary and return its output.
	out, err := b.session.client.run("uname", "-a")
	if err != nil {
		t.Fatalf("exec uname: %v", err)
	}
	if !strings.Contains(string(out), "Linux") {
		t.Fatalf("uname output = %q, want it to contain Linux", out)
	}

	// The blank disk attaches as /dev/vda; format it whole-disk ext4 in the guest.
	if _, err := b.session.client.run("mkfs.ext4", "-F", "-q", "/dev/vda"); err != nil {
		t.Fatalf("mkfs.ext4 /dev/vda: %v", err)
	}

	// Discovery must now see a whole-disk ext4 filesystem.
	parts, err := b.discoverPartitions("/dev/vda")
	if err != nil {
		t.Fatalf("discoverPartitions after mkfs: %v", err)
	}
	if len(parts) != 1 || parts[0].FSType != "ext4" {
		t.Fatalf("discoverPartitions = %+v, want one ext4 filesystem", parts)
	}

	// Mount it and round-trip a file through the backend's guest-path API.
	if err := b.Mount("/dev/vda", mountRoot, "ext4", false); err != nil {
		t.Fatalf("Mount /dev/vda: %v", err)
	}

	want := []byte("hello from the qemu appliance\n")
	if err := b.WriteFile("/greeting.txt", want, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := b.ReadFile("/greeting.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("ReadFile = %q, want %q", got, want)
	}
	if !b.Exists("/greeting.txt") {
		t.Fatalf("Exists(/greeting.txt) = false")
	}

	// StatFS on the mounted filesystem should report positive free space.
	freeBytes, freeInodes, err := b.StatFS("/")
	if err != nil {
		t.Fatalf("StatFS: %v", err)
	}
	if freeBytes <= 0 || freeInodes <= 0 {
		t.Fatalf("StatFS = (%d bytes, %d inodes), want both positive", freeBytes, freeInodes)
	}

	// Unmount cleanly (full Teardown also runs via t.Cleanup).
	if err := b.UnmountFilesystems(); err != nil {
		t.Fatalf("UnmountFilesystems: %v", err)
	}
}
