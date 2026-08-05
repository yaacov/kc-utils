package multipleips

import (
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"
)

func TestShouldRunFalseWhenNoMultipleIPs(t *testing.T) {
	p := &Plugin{}
	cfg := &firstboot.ContributorConfig{
		StaticIPs: []types.StaticIP{
			{MAC: "00:11:22:33:44:55", IP: "10.0.0.1"},
			{MAC: "aa:bb:cc:dd:ee:ff", IP: "10.0.0.2"},
		},
		Options: types.PrepareOptions{MultipleIPsPerNic: true},
	}
	if p.ShouldRun(cfg) {
		t.Error("ShouldRun should be false when each MAC has only one IP")
	}
}

func TestShouldRunTrueWhenMultipleIPs(t *testing.T) {
	p := &Plugin{}
	cfg := &firstboot.ContributorConfig{
		StaticIPs: []types.StaticIP{
			{MAC: "00:11:22:33:44:55", IP: "10.0.0.1", Netmask: "255.255.255.0"},
			{MAC: "00:11:22:33:44:55", IP: "10.0.0.2", Netmask: "255.255.255.0"},
		},
		Options: types.PrepareOptions{MultipleIPsPerNic: true},
	}
	if !p.ShouldRun(cfg) {
		t.Error("ShouldRun should be true when a MAC has multiple IPs")
	}
}

func TestShouldRunFalseWhenOptionDisabled(t *testing.T) {
	p := &Plugin{}
	cfg := &firstboot.ContributorConfig{
		StaticIPs: []types.StaticIP{
			{MAC: "00:11:22:33:44:55", IP: "10.0.0.1"},
			{MAC: "00:11:22:33:44:55", IP: "10.0.0.2"},
		},
		Options: types.PrepareOptions{MultipleIPsPerNic: false},
	}
	if p.ShouldRun(cfg) {
		t.Error("ShouldRun should be false when MultipleIPsPerNic is false")
	}
}

func TestGeneratePowerShell(t *testing.T) {
	p := &Plugin{}
	cfg := &firstboot.ContributorConfig{
		StaticIPs: []types.StaticIP{
			{MAC: "00:11:22:33:44:55", IP: "10.0.0.1", Netmask: "255.255.255.0", Gateway: "10.0.0.254"},
			{MAC: "00:11:22:33:44:55", IP: "10.0.0.2", Netmask: "255.255.255.0"},
			{MAC: "00:11:22:33:44:55", IP: "10.0.0.3", Netmask: "255.255.255.0", DNS: []string{"8.8.8.8"}},
		},
		Options: types.PrepareOptions{MultipleIPsPerNic: true},
	}
	content, err := p.Generate(cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should contain secondary IPs but not the first one
	if strings.Contains(content, `"10.0.0.1"`) {
		t.Error("should not contain the primary IP (10.0.0.1)")
	}
	if !strings.Contains(content, `"10.0.0.2"`) {
		t.Error("should contain secondary IP 10.0.0.2")
	}
	if !strings.Contains(content, `"10.0.0.3"`) {
		t.Error("should contain secondary IP 10.0.0.3")
	}
	if !strings.Contains(content, "New-NetIPAddress") {
		t.Error("should use New-NetIPAddress cmdlet")
	}
	if !strings.Contains(content, "netkvm.sys") {
		t.Error("should wait for netkvm driver")
	}
}

func TestGenerateRegistry(t *testing.T) {
	p := &Plugin{}
	cfg := &firstboot.ContributorConfig{
		StaticIPs: []types.StaticIP{
			{MAC: "00:11:22:33:44:55", IP: "10.0.0.1", Netmask: "255.255.255.0"},
			{MAC: "00:11:22:33:44:55", IP: "10.0.0.2", Netmask: "255.255.255.0"},
		},
		Options: types.PrepareOptions{
			MultipleIPsPerNic:      true,
			WindowsRegistryNetwork: true,
		},
	}
	content, err := p.Generate(cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(content, "Set-ItemProperty") {
		t.Error("registry script should use Set-ItemProperty")
	}
	// Registry mode sets ALL IPs (not just secondary)
	if !strings.Contains(content, "10.0.0.1") {
		t.Error("should contain first IP in registry mode")
	}
	if !strings.Contains(content, "10.0.0.2") {
		t.Error("should contain second IP in registry mode")
	}
}
