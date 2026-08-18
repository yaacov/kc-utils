//go:build unix

package vmware

import (
	"os"
	"path/filepath"
	"strings"
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

	script := filepath.Join(root, "var", "lib", "kc-firstboot", "remove-vmware-pkgs.sh")
	data, err := os.ReadFile(script)
	if err != nil {
		t.Fatalf("pkg-remove script missing: %v", err)
	}
	scriptBody := string(data)
	for _, pkg := range []string{"open-vm-tools", "open-vm-tools-desktop", "VMwareTools"} {
		if !strings.Contains(scriptBody, pkg) {
			t.Errorf("pkg-remove script missing %q", pkg)
		}
	}
	if !strings.Contains(scriptBody, "rpm -e") {
		t.Error("pkg-remove script missing rpm -e")
	}

	unitPath := filepath.Join(root, "etc", "systemd", "system", "kc-remove-vmware.service")
	if _, err := os.Stat(unitPath); err != nil {
		t.Fatalf("kc-remove-vmware.service missing: %v", err)
	}
	wants := filepath.Join(root, "etc", "systemd", "system", "multi-user.target.wants", "kc-remove-vmware.service")
	info, err := os.Lstat(wants)
	if err != nil {
		t.Fatalf("kc-remove-vmware wants symlink missing: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("kc-remove-vmware wants path is not a symlink: %v", info.Mode())
	}
	target, err := os.Readlink(wants)
	if err != nil {
		t.Fatalf("kc-remove-vmware wants symlink unreadable: %v", err)
	}
	if target != "/etc/systemd/system/kc-remove-vmware.service" {
		t.Fatalf("kc-remove-vmware wants symlink target = %q, want /etc/systemd/system/kc-remove-vmware.service", target)
	}
}

func TestDisableVMwareRepos(t *testing.T) {
	root := t.TempDir()
	reposDir := filepath.Join(root, "etc", "yum.repos.d")
	if err := os.MkdirAll(reposDir, 0o755); err != nil {
		t.Fatal(err)
	}

	vmwareRepo := filepath.Join(reposDir, "vmware.repo")
	otherRepo := filepath.Join(reposDir, "fedora.repo")
	if err := os.WriteFile(vmwareRepo, []byte("[vmware]\nbaseurl=https://packages.vmware.com/tools\nenabled=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	otherBody := "[fedora]\nbaseurl=https://download.fedoraproject.org\nenabled=1\n"
	if err := os.WriteFile(otherRepo, []byte(otherBody), 0o644); err != nil {
		t.Fatal(err)
	}

	disableVMwareRepos(root)

	got, err := os.ReadFile(vmwareRepo)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "enabled=0") {
		t.Errorf("vmware repo not disabled: %q", got)
	}
	if strings.Contains(string(got), "enabled=1") {
		t.Errorf("vmware repo still has enabled=1: %q", got)
	}

	other, err := os.ReadFile(otherRepo)
	if err != nil {
		t.Fatal(err)
	}
	if string(other) != otherBody {
		t.Errorf("unrelated repo changed: got %q want %q", other, otherBody)
	}
}
