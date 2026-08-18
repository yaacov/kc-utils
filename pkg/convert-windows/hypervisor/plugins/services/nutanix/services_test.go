//go:build unix

package nutanix

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/registry/mock"
)

func TestDetectPresentByRegistry(t *testing.T) {
	hive := mock.NewMockHive()
	ccs := "ControlSet001"
	hive.CreateKey(ccs + "\\Services\\NutanixGuestTools")

	u := &Services{}
	if !u.Detect(t.TempDir(), hive, ccs) {
		t.Error("Detect = false, want true when NutanixGuestTools service key exists")
	}
}

func TestDetectPresentByDir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Program Files", "Nutanix"), 0o755); err != nil {
		t.Fatal(err)
	}
	hive := mock.NewMockHive()

	u := &Services{}
	if !u.Detect(root, hive, "ControlSet001") {
		t.Error("Detect = false, want true when Nutanix dir exists")
	}
}

func TestDetectPresentX86(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Program Files (x86)", "Nutanix"), 0o755); err != nil {
		t.Fatal(err)
	}
	hive := mock.NewMockHive()

	u := &Services{}
	if !u.Detect(root, hive, "ControlSet001") {
		t.Error("Detect = false, want true when x86 Nutanix dir exists")
	}
}

func TestDetectAbsent(t *testing.T) {
	root := t.TempDir()
	hive := mock.NewMockHive()

	u := &Services{}
	if u.Detect(root, hive, "ControlSet001") {
		t.Error("Detect = true, want false on empty dir")
	}
}

func TestServiceNames(t *testing.T) {
	u := &Services{}
	names := u.ServiceNames()
	if len(names) == 0 {
		t.Fatal("ServiceNames should return at least one name")
	}
	found := false
	for _, n := range names {
		if n == "NutanixGuestTools" {
			found = true
		}
	}
	if !found {
		t.Error("ServiceNames should contain NutanixGuestTools")
	}
}

func TestDisableServicesWritesRegistry(t *testing.T) {
	root := t.TempDir()
	hive := mock.NewMockHive()
	ccs := "ControlSet001"
	hive.CreateKey(ccs + "\\Services\\NutanixGuestTools")

	u := &Services{}
	if err := u.DisableServices(root, hive, ccs); err != nil {
		t.Fatalf("DisableServices error: %v", err)
	}

	start, err := hive.GetDWORD(ccs+"\\Services\\NutanixGuestTools", "Start")
	if err != nil {
		t.Fatalf("GetDWORD error: %v", err)
	}
	if start != 4 {
		t.Errorf("Start = %d, want 4 (disabled)", start)
	}
}

func TestDisableServicesSkipsMissing(t *testing.T) {
	root := t.TempDir()
	hive := mock.NewMockHive()

	u := &Services{}
	if err := u.DisableServices(root, hive, "ControlSet001"); err != nil {
		t.Fatalf("DisableServices error: %v", err)
	}
	if len(hive.Ops) > 0 {
		t.Errorf("expected no ops for missing services, got %d", len(hive.Ops))
	}
}
