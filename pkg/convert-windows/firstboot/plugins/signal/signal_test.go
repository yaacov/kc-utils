package signal

import (
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"
	"github.com/yaacov/kc-utils/pkg/convert-windows/version"
)

func TestShouldRun(t *testing.T) {
	p := &Plugin{}
	if p.ShouldRun(&firstboot.ContributorConfig{}) {
		t.Error("ShouldRun true when WaitForGuestReboot unset")
	}
	if !p.ShouldRun(&firstboot.ContributorConfig{
		Options: types.PrepareOptions{WaitForGuestReboot: true},
	}) {
		t.Error("ShouldRun false when WaitForGuestReboot set")
	}
}

func TestGenerate(t *testing.T) {
	p := &Plugin{}
	content, err := p.Generate(&firstboot.ContributorConfig{
		Options: types.PrepareOptions{WaitForGuestReboot: true},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(content, "CONVERSION_DONE") {
		t.Errorf("script missing CONVERSION_DONE: %q", content)
	}
	if !strings.Contains(content, "Out-Null") {
		t.Errorf("expected PowerShell signal script: %q", content)
	}
	if p.Name() != "signal-conversion-done" {
		t.Errorf("Name = %q", p.Name())
	}
	if p.Priority() != 99999 {
		t.Errorf("Priority = %d", p.Priority())
	}
	if p.UsesBatch(&firstboot.ContributorConfig{}) {
		t.Error("UsesBatch true without version")
	}
}

func TestGenerateBatchForXP(t *testing.T) {
	p := &Plugin{}
	h := version.Classify(&types.InspectData{
		MajorVersion: 5,
		MinorVersion: 1,
		ProductName:  "Windows XP Professional",
	})
	cfg := &firstboot.ContributorConfig{
		Options: types.PrepareOptions{WaitForGuestReboot: true},
		Version: h,
	}
	if !p.UsesBatch(cfg) {
		t.Fatal("XP should use batch signal script")
	}
	content, err := p.Generate(cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(content, "CONVERSION_DONE") {
		t.Errorf("script missing CONVERSION_DONE: %q", content)
	}
	if strings.Contains(content, "Out-Null") {
		t.Errorf("XP signal must be batch, got PowerShell: %q", content)
	}
}
