package driverdb

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/registry/mock"
)

func TestRegisterNonBootCriticalDriver(t *testing.T) {
	h := mock.NewMockHive()
	r := &DriverDBRegistrar{}
	err := r.Register(h, "ControlSet001", "netkvm", `system32\drivers\netkvm.sys`, "NDIS", "x86_64")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	svcPath := `ControlSet001\Services\netkvm`
	if !h.KeyExists(svcPath) {
		t.Fatalf("service key %q was not created", svcPath)
	}

	start, err := h.GetDWORD(svcPath, "Start")
	if err != nil {
		t.Fatalf("GetDWORD Start: %v", err)
	}
	if start != 3 {
		t.Errorf("Start = %d, want 3 (demand-start for non-boot-critical)", start)
	}
}

func TestRegisterBootCriticalDriver(t *testing.T) {
	h := mock.NewMockHive()
	r := &DriverDBRegistrar{}
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
		t.Errorf("Start = %d, want 0 (boot-start for boot-critical)", start)
	}

	ddbPath := `DriverDatabase\DriverInfFiles\viostor.inf`
	if !h.KeyExists(ddbPath) {
		t.Fatalf("DriverDatabase key %q was not created", ddbPath)
	}
	devPath := `DriverDatabase\DeviceIds\PCI\VEN_1AF4&DEV_1001&REV_00`
	if !h.KeyExists(devPath) {
		t.Fatalf("DeviceIds key %q was not created", devPath)
	}
}

func TestRegisterViostorAndVioscsiKeepSeparateServices(t *testing.T) {
	h := mock.NewMockHive()
	r := &DriverDBRegistrar{}

	if err := r.Register(h, "ControlSet001", "viostor", `system32\drivers\viostor.sys`, "SCSI miniport", "x86_64"); err != nil {
		t.Fatalf("viostor: %v", err)
	}
	if err := r.Register(h, "ControlSet001", "vioscsi", `system32\drivers\vioscsi.sys`, "SCSI miniport", "x86_64"); err != nil {
		t.Fatalf("vioscsi: %v", err)
	}

	viostorCfg := `DriverDatabase\DriverPackages\viostor.inf_amd64_0000000000000000\Configurations\viostor_conf`
	svc, err := h.GetString(viostorCfg, "Service")
	if err != nil {
		t.Fatalf("GetString viostor Service: %v", err)
	}
	if svc != "viostor" {
		t.Errorf("viostor Service = %q, want viostor", svc)
	}

	vioscsiCfg := `DriverDatabase\DriverPackages\vioscsi.inf_amd64_0000000000000000\Configurations\vioscsi_conf`
	svc, err = h.GetString(vioscsiCfg, "Service")
	if err != nil {
		t.Fatalf("GetString vioscsi Service: %v", err)
	}
	if svc != "vioscsi" {
		t.Errorf("vioscsi Service = %q, want vioscsi", svc)
	}

	modern := `DriverDatabase\DriverPackages\viostor.inf_amd64_0000000000000000\Configurations\viostor_conf`
	svc, err = h.GetString(modern, "Service")
	if err != nil {
		t.Fatalf("GetString modern viostor Service: %v", err)
	}
	if svc != "viostor" {
		t.Errorf("modern virtio-blk Service = %q, want viostor (not overwritten by vioscsi)", svc)
	}
}

func TestRegisterIdempotentNonBootCritical(t *testing.T) {
	h := mock.NewMockHive()
	r := &DriverDBRegistrar{}

	svcPath := `ControlSet001\Services\netkvm`
	h.CreateKey(svcPath)
	h.SetDWORD(svcPath, "Start", 99)
	opsBeforeRegister := len(h.Ops)

	err := r.Register(h, "ControlSet001", "netkvm", `system32\drivers\netkvm.sys`, "NDIS", "x86_64")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if len(h.Ops) != opsBeforeRegister {
		t.Errorf("Register added %d ops when key already exists, want 0", len(h.Ops)-opsBeforeRegister)
	}
}

func TestRegisterBootCriticalForcesStartZero(t *testing.T) {
	h := mock.NewMockHive()
	r := &DriverDBRegistrar{}

	svcPath := `ControlSet001\Services\viostor`
	h.CreateKey(svcPath)
	h.SetDWORD(svcPath, "Start", 3)

	err := r.Register(h, "ControlSet001", "viostor", `system32\drivers\viostor.sys`, "SCSI miniport", "x86_64")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	start, err := h.GetDWORD(svcPath, "Start")
	if err != nil {
		t.Fatalf("GetDWORD Start: %v", err)
	}
	if start != 0 {
		t.Errorf("Start = %d, want 0 (boot-critical forced to boot-start)", start)
	}
}
