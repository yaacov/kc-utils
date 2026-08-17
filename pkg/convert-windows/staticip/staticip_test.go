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
		got := NetmaskToPrefixLength(mask)
		if got != want {
			t.Errorf("NetmaskToPrefixLength(%q) = %q, want %q", mask, got, want)
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
		"EnableDHCP", "10.0.0.1", "255.255.255.0", "10.0.0.254", "8.8.8.8",
		// The interface subkey must be resolved from the adapter's GUID
		// (SettingID) at boot, matched by the normalized MAC — never keyed by
		// the MAC itself.
		"Win32_NetworkAdapterConfiguration", "SettingID", "525400AABBCC",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("RegistryScript missing %q", want)
		}
	}
	// Regression guard: the MAC must not be used as the interface key.
	if strings.Contains(script, `Interfaces\{52`) || strings.Contains(script, "52-54-00-AA-BB-CC") {
		t.Errorf("RegistryScript keys the interface by MAC instead of SettingID:\n%s", script)
	}
}

func TestRegistryBatScript(t *testing.T) {
	if RegistryBatScript(nil) != "" {
		t.Fatal("empty IPs should yield empty script")
	}
	script := RegistryBatScript([]types.StaticIP{{
		MAC: "52:54:00:aa:bb:cc", IP: "10.0.0.1",
		Netmask: "255.255.255.0", Gateway: "10.0.0.254",
		DNS: []string{"8.8.8.8"},
	}})
	for _, want := range []string{
		"@echo off", "wmic nicconfig", "SettingID", "MACAddress='52:54:00:AA:BB:CC'",
		`Interfaces\!GUID0!`, "EnableDHCP", "10.0.0.1",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("RegistryBatScript missing %q", want)
		}
	}
	if strings.Contains(script, `Interfaces\{52`) {
		t.Errorf("RegistryBatScript keys the interface by MAC instead of SettingID:\n%s", script)
	}
}

func TestWMIScript(t *testing.T) {
	script := WMIScript([]types.StaticIP{{
		MAC: "52:54:00:aa:bb:cc", IP: "10.0.0.1", Netmask: "255.255.255.0",
		DNS: []string{"8.8.8.8"},
	}})
	// The MAC match must strip separators so it works against WMI's
	// colon-separated MACAddress values.
	for _, want := range []string{
		`($_.MACAddress -replace '[^0-9A-Fa-f]','')`, "525400AABBCC", "EnableStatic",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("WMIScript missing %q", want)
		}
	}
}

func TestRebootSignalScript(t *testing.T) {
	script := RebootSignalScript()
	if !strings.Contains(script, "CONVERSION_DONE") {
		t.Errorf("missing CONVERSION_DONE: %q", script)
	}
}

func TestRebootSignalBatScript(t *testing.T) {
	script := RebootSignalBatScript()
	if !strings.Contains(script, "CONVERSION_DONE") {
		t.Errorf("missing CONVERSION_DONE: %q", script)
	}
	if !strings.HasPrefix(script, "@echo off") {
		t.Errorf("expected batch script: %q", script)
	}
}

func TestVMwareCleanupScript(t *testing.T) {
	script := VMwareCleanupScript()
	if !strings.Contains(script, "VMware") {
		t.Errorf("missing VMware: %q", script)
	}
}
