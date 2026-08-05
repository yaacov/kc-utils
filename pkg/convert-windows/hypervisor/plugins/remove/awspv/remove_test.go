//go:build linux

package awspv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/registry/mock"
)

func TestDetectPresent(t *testing.T) {
	h := mock.NewMockHive()
	h.CreateKey(`Microsoft\Windows\CurrentVersion\Uninstall\AWS PV Drivers`)

	u := &Remove{}
	if !u.Detect("/fake", nil, h) {
		t.Error("Detect returned false, want true when AWS PV key exists")
	}
}

func TestDetectViaDriverFiles(t *testing.T) {
	guestRoot := t.TempDir()
	driversDir := filepath.Join(guestRoot, "Windows", "System32", "drivers")
	if err := os.MkdirAll(driversDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(driversDir, "xenvbd.sys"), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := mock.NewMockHive()

	u := &Remove{}
	if !u.Detect(guestRoot, nil, h) {
		t.Error("Detect returned false, want true when xen*.sys driver exists (no registry key)")
	}
}

func TestDetectAbsent(t *testing.T) {
	guestRoot := t.TempDir()
	h := mock.NewMockHive()

	u := &Remove{}
	if u.Detect(guestRoot, nil, h) {
		t.Error("Detect returned true, want false when neither driver files nor registry key exists")
	}
}

func TestRemove(t *testing.T) {
	guestRoot := t.TempDir()
	driversDir := filepath.Join(guestRoot, "Windows", "System32", "drivers")
	if err := os.MkdirAll(driversDir, 0o755); err != nil {
		t.Fatal(err)
	}

	xenFiles := []string{"xenvbd.sys", "xennet.sys", "xenvif.sys"}
	for _, f := range xenFiles {
		if err := os.WriteFile(filepath.Join(driversDir, f), []byte("fake"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Non-xen file should be kept.
	if err := os.WriteFile(filepath.Join(driversDir, "ntfs.sys"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := mock.NewMockHive()
	h.CreateKey(`Microsoft\Windows\CurrentVersion\Uninstall\AWS PV Drivers`)

	u := &Remove{}
	if err := u.Remove(guestRoot, nil, h); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}

	if h.KeyExists(`Microsoft\Windows\CurrentVersion\Uninstall\AWS PV Drivers`) {
		t.Error("uninstall key still exists after Remove")
	}

	for _, f := range xenFiles {
		if _, err := os.Stat(filepath.Join(driversDir, f)); !os.IsNotExist(err) {
			t.Errorf("xen driver %s still exists after Remove", f)
		}
	}

	if _, err := os.Stat(filepath.Join(driversDir, "ntfs.sys")); err != nil {
		t.Error("non-xen driver ntfs.sys was incorrectly removed")
	}
}
