package qemuga

import (
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/convert-windows/driversource"
	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"
)

func TestGenerateUsesExactMSIBasename(t *testing.T) {
	p := &Plugin{}
	cfg := &firstboot.ContributorConfig{
		DriverFiles: []driversource.DriverFile{{
			Name:    "qemu-ga",
			InfPath: "/usr/share/virtio-win/guest-agent/qemu-ga-x86_64.msi",
		}},
	}
	if !p.ShouldRun(cfg) {
		t.Fatal("ShouldRun=false")
	}
	script, err := p.Generate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(script, `C:\Windows\Drivers\VirtIO\qemu-ga-x86_64.msi`) {
		t.Fatalf("script=%s", script)
	}
	if strings.Contains(script, "Get-ChildItem") || strings.Contains(script, "qemu-ga*.msi") {
		t.Fatalf("should not glob MSI: %s", script)
	}
}

func TestShouldRunFalseWithoutMSI(t *testing.T) {
	p := &Plugin{}
	cfg := &firstboot.ContributorConfig{
		DriverFiles: []driversource.DriverFile{{
			Name:    "viostor",
			InfPath: "viostor.inf",
		}},
	}
	if p.ShouldRun(cfg) {
		t.Fatal("expected ShouldRun=false")
	}
}
