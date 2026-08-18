//go:build unix

package virtualbox

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/registry/mock"
)

func TestDetectPresentByRegistry(t *testing.T) {
	hive := mock.NewMockHive()
	ccs := "ControlSet001"
	hive.CreateKey(ccs + "\\Services\\VBoxService")

	u := &Services{}
	if !u.Detect(t.TempDir(), hive, ccs) {
		t.Error("Detect = false, want true when VBoxService key exists")
	}
}

func TestDetectPresentByDir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Program Files", "Oracle", "VirtualBox Guest Additions"), 0o755); err != nil {
		t.Fatal(err)
	}
	hive := mock.NewMockHive()

	u := &Services{}
	if !u.Detect(root, hive, "ControlSet001") {
		t.Error("Detect = false, want true with VirtualBox Guest Additions dir")
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

func TestDisableServicesWritesRegistry(t *testing.T) {
	root := t.TempDir()
	hive := mock.NewMockHive()
	ccs := "ControlSet001"
	hive.CreateKey(ccs + "\\Services\\VBoxService")

	u := &Services{}
	if err := u.DisableServices(root, hive, ccs); err != nil {
		t.Fatalf("DisableServices error: %v", err)
	}

	start, err := hive.GetDWORD(ccs+"\\Services\\VBoxService", "Start")
	if err != nil {
		t.Fatalf("GetDWORD error: %v", err)
	}
	if start != 4 {
		t.Errorf("Start = %d, want 4 (disabled)", start)
	}
}
