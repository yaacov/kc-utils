//go:build linux

package vmware

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor/plugins/testassert"
)

func TestDetectPresent(t *testing.T) {
	root := t.TempDir()
	// Create one of the VMware indicator paths.
	if err := os.MkdirAll(filepath.Join(root, "etc", "vmware-tools"), 0o755); err != nil {
		t.Fatal(err)
	}

	u := &Cleanup{}
	if !u.Detect(root) {
		t.Errorf("Detect = false, want true when vmware-tools dir exists")
	}
}

func TestDetectAbsent(t *testing.T) {
	root := t.TempDir()

	u := &Cleanup{}
	if u.Detect(root) {
		t.Errorf("Detect = true, want false on empty dir")
	}
}

func TestCleanup(t *testing.T) {
	root := t.TempDir()

	// Create the service symlink directories and files that Cleanup removes.
	wantsDir := filepath.Join(root, "etc", "systemd", "system", "multi-user.target.wants")
	if err := os.MkdirAll(wantsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	vendorLibWantsDir := filepath.Join(root, "usr", "lib", "systemd", "system", "multi-user.target.wants")
	if err := os.MkdirAll(vendorLibWantsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	vmtoolsd := filepath.Join(wantsDir, "vmtoolsd.service")
	openvm := filepath.Join(wantsDir, "open-vm-tools.service")
	if err := os.WriteFile(vmtoolsd, []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(openvm, []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorLibWantsDir, "vmtoolsd.service"), []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorLibWantsDir, "open-vm-tools.service"), []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	u := &Cleanup{}
	if err := u.Cleanup(root); err != nil {
		t.Fatalf("Cleanup returned error: %v", err)
	}

	if _, err := os.Stat(vmtoolsd); !os.IsNotExist(err) {
		t.Errorf("vmtoolsd.service still exists after Cleanup")
	}
	if _, err := os.Stat(openvm); !os.IsNotExist(err) {
		t.Errorf("open-vm-tools.service still exists after Cleanup")
	}
	testassert.UnitDisabled(t, root, "vmtoolsd.service")
	testassert.UnitDisabled(t, root, "open-vm-tools.service")
}
