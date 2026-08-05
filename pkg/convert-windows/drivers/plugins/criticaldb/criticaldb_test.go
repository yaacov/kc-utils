package criticaldb

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/registry/mock"
)

func TestRegisterCreatesServiceKey(t *testing.T) {
	h := mock.NewMockHive()
	r := &CriticalDBRegistrar{}
	err := r.Register(h, "ControlSet001", "viostor", `system32\drivers\viostor.sys`, "SCSI miniport", "x86_64")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	svcPath := `ControlSet001\Services\viostor`
	if !h.KeyExists(svcPath) {
		t.Fatalf("service key %q was not created", svcPath)
	}

	if len(h.Ops) == 0 {
		t.Fatal("expected at least one operation recorded")
	}
	if h.Ops[0].Action != "create-key" {
		t.Errorf("first op action = %q, want create-key", h.Ops[0].Action)
	}
	if h.Ops[0].Path != svcPath {
		t.Errorf("first op path = %q, want %q", h.Ops[0].Path, svcPath)
	}
}

func TestRegisterSetsBootStart(t *testing.T) {
	h := mock.NewMockHive()
	r := &CriticalDBRegistrar{}
	err := r.Register(h, "ControlSet001", "viostor", `system32\drivers\viostor.sys`, "SCSI miniport", "x86_64")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	svcPath := `ControlSet001\Services\viostor`
	start, err := h.GetDWORD(svcPath, "Start")
	if err != nil {
		t.Fatalf("GetDWORD Start: %v", err)
	}
	if start != 0 {
		t.Errorf("Start = %d, want 0 (boot-start)", start)
	}
}

func TestRegisterSetsImagePath(t *testing.T) {
	h := mock.NewMockHive()
	r := &CriticalDBRegistrar{}
	driverPath := `system32\drivers\viostor.sys`
	err := r.Register(h, "ControlSet001", "viostor", driverPath, "SCSI miniport", "x86_64")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	svcPath := `ControlSet001\Services\viostor`
	img, err := h.GetString(svcPath, "ImagePath")
	if err != nil {
		t.Fatalf("GetString ImagePath: %v", err)
	}
	if img != driverPath {
		t.Errorf("ImagePath = %q, want %q", img, driverPath)
	}
}

func TestRegisterSetsGroup(t *testing.T) {
	h := mock.NewMockHive()
	r := &CriticalDBRegistrar{}
	group := "SCSI miniport"
	err := r.Register(h, "ControlSet001", "viostor", `system32\drivers\viostor.sys`, group, "x86_64")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	svcPath := `ControlSet001\Services\viostor`
	got, err := h.GetString(svcPath, "Group")
	if err != nil {
		t.Fatalf("GetString Group: %v", err)
	}
	if got != group {
		t.Errorf("Group = %q, want %q", got, group)
	}
}

func TestRegisterCreatesCDBEntry(t *testing.T) {
	h := mock.NewMockHive()
	r := &CriticalDBRegistrar{}
	err := r.Register(h, "ControlSet001", "viostor", `system32\drivers\viostor.sys`, "SCSI miniport", "x86_64")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	cdbPath := `ControlSet001\Control\CriticalDeviceDatabase\PCI#VEN_1AF4&DEV_1001&REV_00`
	if !h.KeyExists(cdbPath) {
		t.Fatalf("CDB key %q was not created", cdbPath)
	}
	modernPath := `ControlSet001\Control\CriticalDeviceDatabase\PCI#VEN_1AF4&DEV_1042&REV_01`
	if !h.KeyExists(modernPath) {
		t.Fatalf("modern CDB key %q was not created", modernPath)
	}

	svc, err := h.GetString(cdbPath, "Service")
	if err != nil {
		t.Fatalf("GetString Service: %v", err)
	}
	if svc != "viostor" {
		t.Errorf("Service = %q, want viostor", svc)
	}
}

func TestRegisterNoCDBForUnknownDriver(t *testing.T) {
	h := mock.NewMockHive()
	r := &CriticalDBRegistrar{}
	err := r.Register(h, "ControlSet001", "customdrv", `system32\drivers\custom.sys`, "Other", "x86_64")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	for k := range h.Keys {
		if len(k) > 30 {
			t.Logf("key: %s", k)
		}
	}
	cdbPrefix := `ControlSet001\Control\CriticalDeviceDatabase\`
	for k := range h.Keys {
		if len(k) > len(cdbPrefix) && k[:len(cdbPrefix)] == cdbPrefix {
			t.Errorf("unexpected CDB key created: %s", k)
		}
	}
}
