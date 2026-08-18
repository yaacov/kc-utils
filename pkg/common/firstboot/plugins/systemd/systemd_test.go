//go:build unix

package systemd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCreatesScript(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "systemd", "system"), 0o755); err != nil {
		t.Fatal(err)
	}

	s := &SystemdFirstBoot{}
	commands := []string{"dnf install -y qemu-guest-agent", "systemctl enable --now qemu-guest-agent"}
	if err := s.Install(root, commands); err != nil {
		t.Fatal(err)
	}

	scriptPath := filepath.Join(root, "usr", "local", "bin", "kc-firstboot.sh")
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("script not created: %v", err)
	}
	script := string(content)
	if !strings.HasPrefix(script, "#!/bin/bash") {
		t.Error("script should start with #!/bin/bash")
	}
	if !strings.Contains(script, "run_with_retry") {
		t.Error("script should contain retry wrapper")
	}
	if !strings.Contains(script, "dnf install -y qemu-guest-agent") {
		t.Error("script should contain dnf install command")
	}
	if !strings.Contains(script, "systemctl enable --now qemu-guest-agent") {
		t.Error("script should contain systemctl enable command")
	}
	if !strings.Contains(script, "systemctl disable kc-firstboot.service") {
		t.Error("script should self-disable on completion")
	}
}

func TestInstallScriptIsExecutable(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "systemd", "system"), 0o755); err != nil {
		t.Fatal(err)
	}

	s := &SystemdFirstBoot{}
	if err := s.Install(root, []string{"echo hello"}); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(root, "usr", "local", "bin", "kc-firstboot.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("script mode = %o, want executable", info.Mode().Perm())
	}
}

func TestInstallCreatesUnit(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "systemd", "system"), 0o755); err != nil {
		t.Fatal(err)
	}

	s := &SystemdFirstBoot{}
	if err := s.Install(root, []string{"echo hello"}); err != nil {
		t.Fatal(err)
	}

	unitPath := filepath.Join(root, "etc", "systemd", "system", "kc-firstboot.service")
	content, err := os.ReadFile(unitPath)
	if err != nil {
		t.Fatalf("unit not created: %v", err)
	}
	unit := string(content)
	if !strings.Contains(unit, "[Service]") {
		t.Error("unit should contain [Service] section")
	}
	if !strings.Contains(unit, "ExecStart=/usr/local/bin/kc-firstboot.sh") {
		t.Error("unit should reference kc-firstboot.sh")
	}
	if !strings.Contains(unit, "network-online.target") {
		t.Error("unit should wait for network")
	}
}

func TestInstallCreatesSymlink(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "systemd", "system"), 0o755); err != nil {
		t.Fatal(err)
	}

	s := &SystemdFirstBoot{}
	if err := s.Install(root, []string{"echo hello"}); err != nil {
		t.Fatal(err)
	}

	symlinkPath := filepath.Join(root, "etc", "systemd", "system", "multi-user.target.wants", "kc-firstboot.service")
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("symlink not created: %v", err)
	}
	if target != "/etc/systemd/system/kc-firstboot.service" {
		t.Errorf("symlink target = %q, want /etc/systemd/system/kc-firstboot.service", target)
	}
}
