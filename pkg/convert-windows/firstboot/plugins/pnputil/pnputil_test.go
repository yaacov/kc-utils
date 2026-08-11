package pnputil

import (
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-windows/driversource"
	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"
	"github.com/yaacov/kc-utils/pkg/convert-windows/version"
)

func TestShouldRun(t *testing.T) {
	p := &Plugin{}
	if p.ShouldRun(&firstboot.ContributorConfig{}) {
		t.Error("ShouldRun true without drivers")
	}
	if !p.ShouldRun(&firstboot.ContributorConfig{
		DriverFiles: []driversource.DriverFile{{Name: "viostor", InfPath: "viostor.inf"}},
	}) {
		t.Error("ShouldRun false with drivers")
	}
}

func TestGeneratePS1(t *testing.T) {
	p := &Plugin{}
	cfg := &firstboot.ContributorConfig{
		DriverFiles: []driversource.DriverFile{{Name: "viostor", InfPath: "viostor.inf"}},
	}
	if p.UsesBatch(cfg) {
		t.Fatal("UsesBatch true without version")
	}
	content, err := p.Generate(cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(content, `pnputil /add-driver "C:\Windows\Drivers\VirtIO\viostor.inf"`) {
		t.Errorf("missing pnputil line: %q", content)
	}
	if strings.Contains(content, "@echo off") {
		t.Errorf("expected PowerShell script: %q", content)
	}
	if p.Name() != "install-virtio-drivers" {
		t.Errorf("Name = %q", p.Name())
	}
	if p.Priority() != 2000 {
		t.Errorf("Priority = %d", p.Priority())
	}
}

func TestSkipXP(t *testing.T) {
	p := &Plugin{}
	h := version.Classify(&types.InspectData{
		MajorVersion: 5,
		MinorVersion: 1,
		ProductName:  "Windows XP Professional",
	})
	cfg := &firstboot.ContributorConfig{
		DriverFiles: []driversource.DriverFile{{Name: "viostor", InfPath: "viostor.inf"}},
		Version:     h,
	}
	if p.ShouldRun(cfg) {
		t.Fatal("XP must skip pnputil; tool is unavailable")
	}
	content, err := p.Generate(cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if strings.Contains(strings.ToLower(content), "pnputil") {
		t.Errorf("XP script must not invoke pnputil: %q", content)
	}
}
