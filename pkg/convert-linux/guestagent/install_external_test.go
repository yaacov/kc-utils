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
)

func TestInstallAmazonLinux2023OnlineSkipsWithoutLocalPackages(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "systemd", "system"), 0o755); err != nil {
		t.Fatal(err)
	}

	guestagent.Install(root, "rpm", "dnf", "x86_64", "amzn", 2023, false)

	scriptPath := filepath.Join(root, "usr", "local", "bin", "kc-firstboot.sh")
	if _, err := os.Stat(scriptPath); err == nil {
		t.Error("amzn 2023 without a local package should not create a firstboot script (no guest-repo package exists)")
	}
}

func TestInstallAmazonLinuxOfflineSkipsWithoutLocalPackages(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "systemd", "system"), 0o755); err != nil {
		t.Fatal(err)
	}

	guestagent.Install(root, "rpm", "dnf", "x86_64", "amzn", 2023, true)

	scriptPath := filepath.Join(root, "usr", "local", "bin", "kc-firstboot.sh")
	if _, err := os.Stat(scriptPath); err == nil {
		t.Error("offline amzn without local packages should not create firstboot script")
	}
}

func TestInstallAmazonLinux2OnlineUsesDnf(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "systemd", "system"), 0o755); err != nil {
		t.Fatal(err)
	}

	guestagent.Install(root, "rpm", "dnf", "x86_64", "amzn", 2, false)

	scriptPath := filepath.Join(root, "usr", "local", "bin", "kc-firstboot.sh")
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("firstboot script not created: %v", err)
	}
	script := string(content)
	if !strings.Contains(script, "dnf install -y qemu-guest-agent") {
		t.Errorf("script should use dnf install, got:\n%s", script)
	}
	if strings.Contains(script, "rpm -ivh") {
		t.Errorf("script should not use local rpm -ivh for online amzn 2, got:\n%s", script)
	}
}

func TestInstallAmazonLinux2OfflineSkips(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "systemd", "system"), 0o755); err != nil {
		t.Fatal(err)
	}

	guestagent.Install(root, "rpm", "dnf", "x86_64", "amzn", 2, true)

	scriptPath := filepath.Join(root, "usr", "local", "bin", "kc-firstboot.sh")
	if _, err := os.Stat(scriptPath); err == nil {
		t.Error("offline amzn 2 without a local el7 package should not create firstboot script")
	}
}
