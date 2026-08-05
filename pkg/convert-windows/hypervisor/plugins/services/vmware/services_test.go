//go:build linux

package vmware

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/registry/mock"
)

func TestDetectPresentByRegistry(t *testing.T) {
	hive := mock.NewMockHive()
	ccs := "ControlSet001"
	hive.CreateKey(ccs + "\\Services\\VMTools")

	u := &Services{}
	if !u.Detect(t.TempDir(), hive, ccs) {
		t.Error("Detect = false, want true when VMTools service key exists")
	}
}

func TestDetectPresentByVGAuth(t *testing.T) {
	hive := mock.NewMockHive()
	ccs := "ControlSet001"
	hive.CreateKey(ccs + "\\Services\\VGAuthService")

	u := &Services{}
	if !u.Detect(t.TempDir(), hive, ccs) {
		t.Error("Detect = false, want true when VGAuthService key exists")
	}
}

func TestDetectAbsent(t *testing.T) {
	hive := mock.NewMockHive()

	u := &Services{}
	if u.Detect(t.TempDir(), hive, "ControlSet001") {
		t.Error("Detect = true, want false when no VMware service keys exist")
	}
}

func TestDetectAfterDirectoryRemoval(t *testing.T) {
	hive := mock.NewMockHive()
	ccs := "ControlSet001"
	hive.CreateKey(ccs + "\\Services\\VMTools")
	hive.CreateKey(ccs + "\\Services\\VGAuthService")

	u := &Services{}
	if !u.Detect(t.TempDir(), hive, ccs) {
		t.Error("Detect = false, want true even without VMware Tools directory (registry-based)")
	}
}

func TestDisableServicesWritesRegistry(t *testing.T) {
	root := t.TempDir()
	hive := mock.NewMockHive()
	ccs := "ControlSet001"
	hive.CreateKey(ccs + "\\Services\\VMTools")
	hive.CreateKey(ccs + "\\Services\\VGAuthService")

	u := &Services{}
	if err := u.DisableServices(root, hive, ccs); err != nil {
		t.Fatalf("DisableServices error: %v", err)
	}

	for _, svc := range []string{"VMTools", "VGAuthService"} {
		start, err := hive.GetDWORD(ccs+"\\Services\\"+svc, "Start")
		if err != nil {
			t.Fatalf("GetDWORD(%s) error: %v", svc, err)
		}
		if start != 4 {
			t.Errorf("%s Start = %d, want 4 (disabled)", svc, start)
		}
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
