//go:build unix

package nutanix

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
	if !u.Detect("/fake/root", nil, h) {
		t.Error("Detect returned false, want true when Nutanix Guest Tools key exists")
	}
}

func TestDetectViaDirectory(t *testing.T) {
	guestRoot := t.TempDir()
	nutanixDir := filepath.Join(guestRoot, "Program Files", "Nutanix")
	if err := os.MkdirAll(nutanixDir, 0o755); err != nil {
		t.Fatal(err)
	}

	h := mock.NewMockHive()

	u := &Remove{}
	if !u.Detect(guestRoot, nil, h) {
		t.Error("Detect returned false, want true when Nutanix directory exists (no registry key)")
	}
}

func TestDetectViaDirectoryX86(t *testing.T) {
	guestRoot := t.TempDir()
	nutanixDir := filepath.Join(guestRoot, "Program Files (x86)", "Nutanix")
	if err := os.MkdirAll(nutanixDir, 0o755); err != nil {
		t.Fatal(err)
	}

	h := mock.NewMockHive()

	u := &Remove{}
	if !u.Detect(guestRoot, nil, h) {
		t.Error("Detect returned false, want true when Nutanix x86 directory exists (no registry key)")
	}
}

func TestDetectAbsent(t *testing.T) {
	guestRoot := t.TempDir()
	h := mock.NewMockHive()

	u := &Remove{}
	if u.Detect(guestRoot, nil, h) {
		t.Error("Detect returned true, want false when neither directory nor registry key exists")
	}
}

func TestRemove(t *testing.T) {
	guestRoot := t.TempDir()
	toolsDir := filepath.Join(guestRoot, "Program Files", "Nutanix", "Guest Tools")
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(toolsDir, "ngt.exe")
	if err := os.WriteFile(marker, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := mock.NewMockHive()
	h.CreateKey(uninstallKey)

	u := &Remove{}
	if err := u.Remove(guestRoot, nil, h); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(guestRoot, "Program Files", "Nutanix")); !os.IsNotExist(err) {
		t.Error("Nutanix install dir still exists after Remove")
	}
	if h.KeyExists(uninstallKey) {
		t.Error("Nutanix Guest Tools registry key still exists after Remove")
	}
}
