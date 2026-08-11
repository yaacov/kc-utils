package diskonliner

import (
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"
	"github.com/yaacov/kc-utils/pkg/convert-windows/version"
)

func TestShouldRun(t *testing.T) {
	p := &Plugin{}
	if !p.ShouldRun(&firstboot.ContributorConfig{}) {
		t.Error("ShouldRun false without version")
	}
	xp := version.Classify(&types.InspectData{
		MajorVersion: 5,
		MinorVersion: 1,
		ProductName:  "Windows XP Professional",
	})
	if p.ShouldRun(&firstboot.ContributorConfig{Version: xp}) {
		t.Error("XP DiskOnlineSkip should not run")
	}
}

func TestUsesBatchAndGenerateWin2003(t *testing.T) {
	p := &Plugin{}
	h := version.Classify(&types.InspectData{
		MajorVersion: 5,
		MinorVersion: 2,
		ProductName:  "Windows Server 2003",
	})
	cfg := &firstboot.ContributorConfig{Version: h}
	if !p.UsesBatch(cfg) {
		t.Fatal("Win2003 should use batch")
	}
	content, err := p.Generate(cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(content, "diskpart /s") {
		t.Errorf("expected batch diskpart script: %q", content)
	}
	if strings.Contains(content, "Get-WmiObject") {
		t.Errorf("Win2003 must not emit PowerShell: %q", content)
	}
	if !strings.Contains(content, "select volume") {
		t.Errorf("legacy script must select each volume: %q", content)
	}
	if !strings.Contains(content, "attributes volume clear readonly") {
		t.Errorf("legacy script must clear volume readonly: %q", content)
	}
	if strings.Contains(strings.ToLower(content), "attributes disk") {
		t.Errorf("legacy script must not use ATTRIBUTES DISK: %q", content)
	}
}

func TestGeneratePowerShellWin7(t *testing.T) {
	p := &Plugin{}
	h := version.Classify(&types.InspectData{
		MajorVersion: 6,
		MinorVersion: 1,
		ProductName:  "Windows 7 Professional",
	})
	cfg := &firstboot.ContributorConfig{Version: h}
	if p.UsesBatch(cfg) {
		t.Fatal("Win7 should use PowerShell")
	}
	content, err := p.Generate(cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(content, "Get-WmiObject") {
		t.Errorf("expected WMI diskpart script: %q", content)
	}
	if strings.Contains(content, "@echo off") {
		t.Errorf("Win7 must not emit batch: %q", content)
	}
}

func TestGenerateGetDiskDefault(t *testing.T) {
	p := &Plugin{}
	content, err := p.Generate(&firstboot.ContributorConfig{})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(content, "Get-Disk") {
		t.Errorf("expected Get-Disk script: %q", content)
	}
	if p.UsesBatch(&firstboot.ContributorConfig{}) {
		t.Error("UsesBatch true without version")
	}
	if p.Name() != "disk-onliner" {
		t.Errorf("Name = %q", p.Name())
	}
	if p.Priority() != 4000 {
		t.Errorf("Priority = %d", p.Priority())
	}
}
