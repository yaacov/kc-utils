package staticipfb

import (
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"
)

func TestShouldRun(t *testing.T) {
	p := &Plugin{}
	if p.ShouldRun(&firstboot.ContributorConfig{}) {
		t.Error("ShouldRun true with no static IPs")
	}
	if !p.ShouldRun(&firstboot.ContributorConfig{
		StaticIPs: []types.StaticIP{{MAC: "52:54:00:aa:bb:cc", IP: "10.0.0.1"}},
	}) {
		t.Error("ShouldRun false with static IPs")
	}
}

func TestGeneratePowerShell(t *testing.T) {
	p := &Plugin{}
	cfg := &firstboot.ContributorConfig{
		StaticIPs: []types.StaticIP{{MAC: "52:54:00:aa:bb:cc", IP: "10.0.0.1", Netmask: "255.255.255.0"}},
	}
	content, err := p.Generate(cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(content, "netkvm") && !strings.Contains(content, "VirtIO") {
		t.Error("expected virtio-net wait loop")
	}
	if !strings.Contains(content, "New-NetIPAddress") {
		t.Error("expected PowerShell New-NetIPAddress")
	}
	if strings.Contains(content, "Set-ItemProperty") {
		t.Error("PowerShell path should not use registry Set-ItemProperty")
	}
}

func TestGenerateRegistry(t *testing.T) {
	p := &Plugin{}
	cfg := &firstboot.ContributorConfig{
		StaticIPs: []types.StaticIP{{MAC: "52:54:00:aa:bb:cc", IP: "10.0.0.1", Netmask: "255.255.255.0"}},
		Options:   types.PrepareOptions{WindowsRegistryNetwork: true},
	}
	content, err := p.Generate(cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(content, "Set-ItemProperty") {
		t.Error("expected registry Set-ItemProperty")
	}
	if !strings.Contains(content, "10.0.0.1") {
		t.Error("expected IP in script")
	}
}

func TestGenerateEmpty(t *testing.T) {
	p := &Plugin{}
	content, err := p.Generate(&firstboot.ContributorConfig{})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if content != "" {
		t.Errorf("expected empty content, got %q", content)
	}
}
