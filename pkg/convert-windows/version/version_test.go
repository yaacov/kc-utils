package version_test

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-windows/version"
)

func TestClassifyWin2008RegisteredProductName(t *testing.T) {
	h := version.Classify(&types.InspectData{
		MajorVersion: 6,
		MinorVersion: 0,
		ProductName:  "Windows Server (R) 2008 Enterprise\x00",
	})
	if h.Name() != "win2008" {
		t.Fatalf("got handler %q, want win2008", h.Name())
	}
	if prefs := h.DriverOSPreferences(); len(prefs) != 1 || prefs[0] != "2k8" {
		t.Fatalf("prefs = %v, want [2k8]", prefs)
	}
	if fb := h.DriverOSFallbacks(); len(fb) != 1 || fb[0] != "2k8R2" {
		t.Fatalf("fallbacks = %v, want [2k8R2]", fb)
	}
}

func TestClassifyWin2008R2(t *testing.T) {
	h := version.Classify(&types.InspectData{
		MajorVersion: 6,
		MinorVersion: 1,
		ProductName:  "Windows Server 2008 R2 Standard",
	})
	if h.Name() != "win2008r2" {
		t.Fatalf("got handler %q, want win2008r2", h.Name())
	}
}

func TestClassifyWin7Client(t *testing.T) {
	h := version.Classify(&types.InspectData{
		MajorVersion: 6,
		MinorVersion: 1,
		ProductName:  "Windows 7 Professional",
	})
	if h.Name() != "win7" {
		t.Fatalf("got handler %q, want win7", h.Name())
	}
}

func TestClassifyWin10(t *testing.T) {
	h := version.Classify(&types.InspectData{
		MajorVersion: 10,
		MinorVersion: 0,
		ProductName:  "Windows Server 2019 Standard",
	})
	if h.Name() != "win10" {
		t.Fatalf("got handler %q, want win10", h.Name())
	}
}

func TestClassifyWinXP(t *testing.T) {
	h := version.Classify(&types.InspectData{
		MajorVersion: 5,
		MinorVersion: 1,
		ProductName:  "Microsoft Windows XP Professional",
	})
	if h.Name() != "winxp" {
		t.Fatalf("got handler %q, want winxp", h.Name())
	}
	if h.SupportsPowerShell() {
		t.Fatal("winxp should not support PowerShell firstboot")
	}
}

func TestClassifyWin2003(t *testing.T) {
	h := version.Classify(&types.InspectData{
		MajorVersion: 5,
		MinorVersion: 2,
		ProductName:  "Windows Server 2003 Standard",
	})
	if h.Name() != "win2003" {
		t.Fatalf("got handler %q, want win2003", h.Name())
	}
}

func TestClassifyWin11(t *testing.T) {
	h := version.Classify(&types.InspectData{
		MajorVersion: 10,
		MinorVersion: 0,
		ProductName:  "Windows 11 Pro",
	})
	if h.Name() != "win11" {
		t.Fatalf("got handler %q, want win11", h.Name())
	}
}

func TestDriverRegistrarName(t *testing.T) {
	tests := []struct {
		name    string
		inspect types.InspectData
		want    string
	}{
		{"win10 uses driverdb", types.InspectData{MajorVersion: 10, ProductName: "Windows 10 Pro"}, "driverdb"},
		{"win8 uses driverdb", types.InspectData{MajorVersion: 6, MinorVersion: 2, ProductName: "Windows 8"}, "driverdb"},
		{"win7 uses criticaldb", types.InspectData{MajorVersion: 6, MinorVersion: 1, ProductName: "Windows 7 Professional"}, "criticaldb"},
		{"win2008 uses criticaldb", types.InspectData{MajorVersion: 6, MinorVersion: 0, ProductName: "Windows Server 2008 Enterprise"}, "criticaldb"},
		{"winxp uses criticaldb", types.InspectData{MajorVersion: 5, MinorVersion: 1, ProductName: "Windows XP"}, "criticaldb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := version.Classify(&tt.inspect)
			if got := h.DriverRegistrarName(); got != tt.want {
				t.Fatalf("handler %q: DriverRegistrarName() = %q, want %q", h.Name(), got, tt.want)
			}
		})
	}
}

func TestNeedsNTFSHeadsFix(t *testing.T) {
	tests := []struct {
		name    string
		inspect types.InspectData
		want    bool
	}{
		{"win10 no fix", types.InspectData{MajorVersion: 10, ProductName: "Windows 10 Pro"}, false},
		{"win7 no fix", types.InspectData{MajorVersion: 6, MinorVersion: 1, ProductName: "Windows 7 Professional"}, false},
		{"winvista no fix", types.InspectData{MajorVersion: 6, MinorVersion: 0, ProductName: "Windows Vista"}, false},
		{"win2003 needs fix", types.InspectData{MajorVersion: 5, MinorVersion: 2, ProductName: "Windows Server 2003 Standard"}, true},
		{"winxp needs fix", types.InspectData{MajorVersion: 5, MinorVersion: 1, ProductName: "Windows XP"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := version.Classify(&tt.inspect)
			if got := h.NeedsNTFSHeadsFix(); got != tt.want {
				t.Fatalf("handler %q: NeedsNTFSHeadsFix() = %v, want %v", h.Name(), got, tt.want)
			}
		})
	}
}
