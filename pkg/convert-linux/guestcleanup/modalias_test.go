//go:build linux

package guestcleanup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigureCreatesVirtioConf(t *testing.T) {
	root := t.TempDir()
	Configure(root)

	data, err := os.ReadFile(filepath.Join(root, "etc", "modprobe.d", "kc-virtio.conf"))
	if err != nil {
		t.Fatalf("reading kc-virtio.conf: %v", err)
	}
	content := string(data)
	for _, expect := range []string{"virtio_blk", "virtio_scsi", "virtio_net"} {
		if !strings.Contains(content, expect) {
			t.Errorf("expected %q in kc-virtio.conf", expect)
		}
	}
}

func TestCleanStaleAliases(t *testing.T) {
	root := t.TempDir()
	modprobeDir := filepath.Join(root, "etc", "modprobe.d")
	if err := os.MkdirAll(modprobeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	staleConf := "alias scsi_hostadapter vmw_pvscsi\nalias eth0 vmxnet3\nalias other_thing some_module\n"
	if err := os.WriteFile(filepath.Join(modprobeDir, "vmware.conf"), []byte(staleConf), 0o644); err != nil {
		t.Fatal(err)
	}

	Configure(root)

	data, err := os.ReadFile(filepath.Join(modprobeDir, "vmware.conf"))
	if err != nil {
		t.Fatalf("reading vmware.conf: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "vmw_pvscsi") {
		t.Error("stale vmw_pvscsi alias was not removed")
	}
	if strings.Contains(content, "vmxnet3") {
		t.Error("stale vmxnet3 alias was not removed")
	}
	if !strings.Contains(content, "some_module") {
		t.Error("non-stale alias was incorrectly removed")
	}
}

func TestCleanStaleSkipsKcVirtioConf(t *testing.T) {
	root := t.TempDir()
	modprobeDir := filepath.Join(root, "etc", "modprobe.d")
	if err := os.MkdirAll(modprobeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	Configure(root)

	data, _ := os.ReadFile(filepath.Join(modprobeDir, "kc-virtio.conf"))
	if !strings.Contains(string(data), "virtio_blk") {
		t.Error("kc-virtio.conf should not be cleaned of virtio aliases")
	}
}

func TestRemoveStaleLines(t *testing.T) {
	input := "alias foo vmw_pvscsi\nalias bar virtio_blk\noptions vmxnet3 param=1\n"
	result, changed := removeStaleLines(input)
	if !changed {
		t.Error("expected changed=true")
	}
	if strings.Contains(result, "vmw_pvscsi") {
		t.Error("vmw_pvscsi line should be removed")
	}
	if strings.Contains(result, "vmxnet3") {
		t.Error("vmxnet3 options line should be removed")
	}
	if !strings.Contains(result, "virtio_blk") {
		t.Error("virtio_blk line should be preserved")
	}
}

func TestRemoveStaleLinesNoChange(t *testing.T) {
	input := "alias foo virtio_blk\nalias bar virtio_net\n"
	_, changed := removeStaleLines(input)
	if changed {
		t.Error("expected changed=false for non-stale content")
	}
}
