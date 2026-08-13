//go:build linux

package guestagent_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/yaacov/kc-utils/pkg/common/firstboot/plugins/systemd"
	"github.com/yaacov/kc-utils/pkg/convert-linux/guestagent"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/guestagent/plugins/agent/qemuga"
	"github.com/yaacov/kc-utils/pkg/convert-linux/systemd"
)

func writeQEMUBinary(t *testing.T, root string) {
	t.Helper()
	binDir := filepath.Join(root, "usr", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "qemu-ga"), []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeQEMUUnitFile(t *testing.T, root string) {
	t.Helper()
	unitDir := filepath.Join(root, "usr", "lib", "systemd", "system")
	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unitDir, "qemu-guest-agent.service"), []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func enableQEMUUnitVendor(t *testing.T, root string) {
	t.Helper()
	wantsDir := filepath.Join(root, "usr", "lib", "systemd", "system", "multi-user.target.wants")
	if err := os.MkdirAll(wantsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/usr/lib/systemd/system/qemu-guest-agent.service", filepath.Join(wantsDir, "qemu-guest-agent.service")); err != nil {
		t.Fatal(err)
	}
}

func maskQEMUUnit(t *testing.T, root string) {
	t.Helper()
	maskDir := filepath.Join(root, "etc", "systemd", "system")
	if err := os.MkdirAll(maskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(systemd.UnitMaskTarget, filepath.Join(maskDir, "qemu-guest-agent.service")); err != nil {
		t.Fatal(err)
	}
}

func firstbootScript(t *testing.T, root string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, "usr", "local", "bin", "kc-firstboot.sh"))
	if err != nil {
		return ""
	}
	return string(content)
}

func TestInstallPreinstalledEnabledSkips(t *testing.T) {
	root := t.TempDir()
	writeQEMUBinary(t, root)
	writeQEMUUnitFile(t, root)
	enableQEMUUnitVendor(t, root)

	guestagent.Install(root, "rpm", "dnf", "x86_64", "rhel", 9, false)

	if script := firstbootScript(t, root); script != "" {
		t.Errorf("enabled preinstalled agent should not create firstboot script, got:\n%s", script)
	}
}

func TestInstallPreinstalledDisabledEnablesOffline(t *testing.T) {
	root := t.TempDir()
	writeQEMUBinary(t, root)
	writeQEMUUnitFile(t, root)

	guestagent.Install(root, "rpm", "dnf", "x86_64", "rhel", 9, false)

	if !systemd.UnitWantsEnabled(root, "qemu-guest-agent.service") {
		t.Error("disabled preinstalled agent should be enabled offline")
	}
	if script := firstbootScript(t, root); script != "" {
		t.Errorf("offline enable should not create firstboot script, got:\n%s", script)
	}
}

func TestInstallPreinstalledMaskedEnablesOffline(t *testing.T) {
	root := t.TempDir()
	writeQEMUBinary(t, root)
	writeQEMUUnitFile(t, root)
	maskQEMUUnit(t, root)

	guestagent.Install(root, "rpm", "dnf", "x86_64", "rhel", 9, false)

	if systemd.UnitIsMasked(root, "qemu-guest-agent.service") {
		t.Error("masked preinstalled agent should be unmasked offline")
	}
	if !systemd.UnitWantsEnabled(root, "qemu-guest-agent.service") {
		t.Error("masked preinstalled agent should be enabled offline")
	}
}

func TestInstallPreinstalledWithoutUnitFileUsesFirstboot(t *testing.T) {
	root := t.TempDir()
	writeQEMUBinary(t, root)
	if err := os.MkdirAll(filepath.Join(root, "etc", "systemd", "system"), 0o755); err != nil {
		t.Fatal(err)
	}

	guestagent.Install(root, "rpm", "dnf", "x86_64", "rhel", 9, false)

	script := firstbootScript(t, root)
	if script == "" {
		t.Fatal("missing unit file should schedule firstboot enable commands")
	}
	if !strings.Contains(script, "systemctl unmask qemu-guest-agent") {
		t.Errorf("firstboot script should unmask guest agent, got:\n%s", script)
	}
	if !strings.Contains(script, "systemctl enable --now qemu-guest-agent") {
		t.Errorf("firstboot script should enable guest agent, got:\n%s", script)
	}
	if strings.Contains(script, "dnf install -y qemu-guest-agent") {
		t.Errorf("firstboot script should not install package when binary exists, got:\n%s", script)
	}
}
