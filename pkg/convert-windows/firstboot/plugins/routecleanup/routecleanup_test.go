package routecleanup

import (
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"
	"github.com/yaacov/kc-utils/pkg/convert-windows/version"
)

func TestShouldRun(t *testing.T) {
	p := &Plugin{}
	staticIPs := []types.StaticIP{{MAC: "52:54:00:aa:bb:cc", IP: "10.0.0.1"}}

	win10 := version.Classify(&types.InspectData{
		MajorVersion: 10,
		MinorVersion: 0,
		ProductName:  "Windows 10 Pro",
	})
	vista := version.Classify(&types.InspectData{
		MajorVersion: 6,
		MinorVersion: 0,
		ProductName:  "Windows Vista Business",
	})
	xp := version.Classify(&types.InspectData{
		MajorVersion: 5,
		MinorVersion: 1,
		ProductName:  "Windows XP Professional",
	})

	cases := []struct {
		name string
		cfg  *firstboot.ContributorConfig
		want bool
	}{
		{
			name: "offline",
			cfg:  &firstboot.ContributorConfig{Offline: true, StaticIPs: staticIPs, Version: win10},
			want: false,
		},
		{
			name: "no static IPs",
			cfg:  &firstboot.ContributorConfig{Version: win10},
			want: false,
		},
		{
			name: "WMINetsh mode",
			cfg:  &firstboot.ContributorConfig{StaticIPs: staticIPs, Version: vista},
			want: false,
		},
		{
			name: "no PowerShell",
			cfg:  &firstboot.ContributorConfig{StaticIPs: staticIPs, Version: xp},
			want: false,
		},
		{
			name: "modern with static IPs",
			cfg:  &firstboot.ContributorConfig{StaticIPs: staticIPs, Version: win10},
			want: true,
		},
		{
			name: "nil version with static IPs",
			cfg:  &firstboot.ContributorConfig{StaticIPs: staticIPs},
			want: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := p.ShouldRun(tc.cfg); got != tc.want {
				t.Errorf("ShouldRun = %v, want %v", got, tc.want)
			}
		})
	}
	if p.Name() != "remove-duplicate-routes" {
		t.Errorf("Name = %q", p.Name())
	}
	if p.Priority() != 2600 {
		t.Errorf("Priority = %d", p.Priority())
	}
	if p.UsesBatch(&firstboot.ContributorConfig{}) {
		t.Error("UsesBatch should be false")
	}
}

func TestGenerateDefault(t *testing.T) {
	p := &Plugin{}
	content, err := p.Generate(&firstboot.ContributorConfig{})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(content, "Remove-NetRoute") {
		t.Errorf("default script missing Remove-NetRoute: %q", content)
	}
	if strings.Contains(content, "PersistentRoutes") {
		t.Errorf("default script should not clean PersistentRoutes registry: %q", content)
	}
}

func TestGenerateRegistry(t *testing.T) {
	p := &Plugin{}
	content, err := p.Generate(&firstboot.ContributorConfig{
		Options: types.PrepareOptions{WindowsRegistryNetwork: true},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(content, "PersistentRoutes") {
		t.Errorf("registry script missing PersistentRoutes: %q", content)
	}
	if !strings.Contains(content, "Remove-ItemProperty") {
		t.Errorf("registry script missing Remove-ItemProperty: %q", content)
	}
}
