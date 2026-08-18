//go:build unix

package virtualbox

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/registry/mock"
)

func TestDetectPresent(t *testing.T) {
	h := mock.NewMockHive()
	h.CreateKey(uninstallKey)

	u := &Remove{}
	if !u.Detect("/fake", nil, h) {
		t.Error("Detect = false, want true when uninstall key exists")
	}
}

func TestDetectViaGuestAdditionsDir(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "Program Files", "Oracle", "VirtualBox Guest Additions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	u := &Remove{}
	if !u.Detect(root, nil, mock.NewMockHive()) {
		t.Error("Detect = false, want true for Guest Additions dir")
	}
}

func TestRemove(t *testing.T) {
	root := t.TempDir()
	gaDir := filepath.Join(root, "Program Files", "Oracle", "VirtualBox Guest Additions")
	if err := os.MkdirAll(gaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gaDir, "VBoxService.exe"), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	driversDir := filepath.Join(root, "Windows", "System32", "drivers")
	if err := os.MkdirAll(driversDir, 0o755); err != nil {
		t.Fatal(err)
	}
	vboxSys := filepath.Join(driversDir, "VBoxGuest.sys")
	keepSys := filepath.Join(driversDir, "disk.sys")
	if err := os.WriteFile(vboxSys, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keepSys, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	softwareHive := mock.NewMockHive()
	softwareHive.CreateKey(uninstallKey)

	u := &Remove{}
	if err := u.Remove(root, nil, softwareHive); err != nil {
		t.Fatalf("Remove error: %v", err)
	}

	if _, err := os.Stat(gaDir); !os.IsNotExist(err) {
		t.Error("VirtualBox Guest Additions directory still exists")
	}
	if softwareHive.KeyExists(uninstallKey) {
		t.Error("uninstall key still exists")
	}
	if _, err := os.Stat(vboxSys); !os.IsNotExist(err) {
		t.Error("VBoxGuest.sys still exists")
	}
	if _, err := os.Stat(keepSys); err != nil {
		t.Error("unrelated disk.sys should remain")
	}
}
