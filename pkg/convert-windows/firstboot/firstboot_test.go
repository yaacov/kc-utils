//go:build linux

package firstboot_test

import (
	"os"
	"path/filepath"
	"testing"

	regmock "github.com/yaacov/kc-utils/pkg/common/registry/mock"
	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"

	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/diskonliner"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/multipleips"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/pnputil"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/qemuga"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/routecleanup"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/signal"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/staticipfb"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/vmwarecleanup"
)

func TestWriteScript(t *testing.T) {
	tmpDir := t.TempDir()
	content := "# test script\r\necho hello\r\n"
	err := firstboot.WriteScript(tmpDir, 2000, "test-script", content)
	if err != nil {
		t.Fatalf("WriteScript returned error: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "scripts", "2000-test-script.ps1")
	info, err := os.Stat(expectedPath)
	if err != nil {
		t.Fatalf("expected file %s to exist, got error: %v", expectedPath, err)
	}
	if info.Size() == 0 {
		t.Errorf("expected non-empty file, got size 0")
	}

	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("reading script file: %v", err)
	}
	if string(data) != content {
		t.Errorf("file content mismatch: got %q, want %q", string(data), content)
	}
}

func TestConfigureOfflineStillAddsQemuGAScript(t *testing.T) {
	root := t.TempDir()
	hive := regmock.NewMockHive()

	err := firstboot.Configure(&firstboot.Config{
		MountRoot: root,
		Offline:   true,
	}, hive)
	if err != nil {
		t.Fatalf("Configure returned error: %v", err)
	}

	scriptPath := filepath.Join(root, "Program Files", "Guestfs", "Firstboot", "scripts", "3000-install-qemu-ga.ps1")
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("expected qemu-ga script to exist: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected qemu-ga script to be non-empty")
	}
}
