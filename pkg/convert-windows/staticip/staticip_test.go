package staticip

import (
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestNetmaskToPrefixLength(t *testing.T) {
	cases := map[string]string{
		"255.255.255.0": "24",
		"255.255.0.0":   "16",
		"255.0.0.0":     "8",
		"invalid":       "24",
	}
	for mask, want := range cases {
		got := netmaskToPrefixLength(mask)
		if got != want {
			t.Errorf("netmaskToPrefixLength(%q) = %q, want %q", mask, got, want)
		}
	}
}

func TestPowerShellScript(t *testing.T) {
	script := PowerShellScript([]types.StaticIP{{MAC: "52:54:00:aa:bb:cc", IP: "10.0.0.1"}})
	if script == "" {
		t.Fatal("expected script content")
	}
}

func TestRegistryScript(t *testing.T) {
	if RegistryScript(nil) != "" {
		t.Fatal("empty IPs should yield empty script")
	}
	script := RegistryScript([]types.StaticIP{{
		MAC: "52:54:00:aa:bb:cc", IP: "10.0.0.1",
		Netmask: "255.255.255.0", Gateway: "10.0.0.254",
		DNS: []string{"8.8.8.8"},
	}})
	if script == "" {
		t.Fatal("expected script content")
	}
	for _, want := range []string{
		"EnableDHCP", "10.0.0.1", "255.255.255.0", "10.0.0.254", "8.8.8.8", "52-54-00-AA-BB-CC",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("RegistryScript missing %q", want)
		}
	}
}

func TestRebootSignalScript(t *testing.T) {
	script := RebootSignalScript()
	if !strings.Contains(script, "CONVERSION_DONE") {
		t.Errorf("missing CONVERSION_DONE: %q", script)
	}
}

func TestVMwareCleanupScript(t *testing.T) {
	script := VMwareCleanupScript()
	if !strings.Contains(script, "VMware") {
		t.Errorf("missing VMware: %q", script)
	}
}
