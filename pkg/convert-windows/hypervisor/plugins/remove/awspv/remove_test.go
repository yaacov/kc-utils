//go:build unix

package awspv

import (
	"os"
	"path/filepath"
	"strings"
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

	xenFiles := []string{"xenvbd.sys", "xennet.sys", "xenvif.sys", "xenfilt.sys"}
	for _, f := range xenFiles {
		if err := os.WriteFile(filepath.Join(driversDir, f), []byte("fake"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Non-xen file should be kept.
	if err := os.WriteFile(filepath.Join(driversDir, "ntfs.sys"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	systemHive := mock.NewMockHive()
	systemHive.SetDWORD(`Select`, "Current", 1)
	ccs := "ControlSet001"
	for _, guid := range []string{systemClassGUID, hdcClassGUID} {
		classPath := ccs + `\Control\Class\` + guid
		systemHive.CreateKey(classPath)
		systemHive.SetMultiString(classPath, "UpperFilters", []string{"PartMgr", "XENFILT"})
	}
	systemHive.CreateKey(ccs + `\Services\xenfilt`)

	softwareHive := mock.NewMockHive()
	softwareHive.CreateKey(`Microsoft\Windows\CurrentVersion\Uninstall\AWS PV Drivers`)

	u := &Remove{}
	if err := u.Remove(guestRoot, systemHive, softwareHive); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}

	if softwareHive.KeyExists(`Microsoft\Windows\CurrentVersion\Uninstall\AWS PV Drivers`) {
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

	for _, guid := range []string{systemClassGUID, hdcClassGUID} {
		classPath := ccs + `\Control\Class\` + guid
		filters, err := systemHive.GetMultiString(classPath, "UpperFilters")
		if err != nil {
			t.Fatalf("GetMultiString %s: %v", guid, err)
		}
		for _, f := range filters {
			if strings.EqualFold(f, "XENFILT") {
				t.Errorf("XENFILT still in UpperFilters for %s", guid)
			}
		}
	}

	start, err := systemHive.GetDWORD(ccs+`\Services\xenfilt`, "Start")
	if err != nil {
		t.Fatal(err)
	}
	if start != 4 {
		t.Errorf("xenfilt Start = %d, want 4", start)
	}
}
