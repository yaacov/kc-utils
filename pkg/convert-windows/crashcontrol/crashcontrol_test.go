package crashcontrol

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/registry/mock"
)

func TestDisable(t *testing.T) {
	hive := mock.NewMockHive()
	ccs := "ControlSet001"

	Disable(hive, ccs)

	path := ccs + `\Control\CrashControl`
	if !hive.KeyExists(path) {
		t.Fatal("expected CrashControl key to be created")
	}
	val, err := hive.GetDWORD(path, "AutoReboot")
	if err != nil {
		t.Fatalf("GetDWORD: %v", err)
	}
	if val != 0 {
		t.Errorf("AutoReboot = %d, want 0", val)
	}
}
