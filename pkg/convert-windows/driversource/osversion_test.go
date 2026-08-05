package driversource

import "testing"

func TestCanonicalOSVersionsWin2008ProductName(t *testing.T) {
	aliases := CanonicalOSVersions("Windows Server (R) 2008 Enterprise\x00")
	has2k8 := false
	for _, a := range aliases {
		if a == "2k8" {
			has2k8 = true
			break
		}
	}
	if !has2k8 {
		t.Fatalf("expected 2k8 alias in %v", aliases)
	}
}

func TestMatchOSVersionAliases(t *testing.T) {
	tests := []struct {
		dirVer     string
		requested  string
		shouldFind bool
	}{
		{dirVer: "2k8R2", requested: "Windows Server (R) 2008 Enterprise\x00", shouldFind: false},
		{dirVer: "2k22", requested: "Windows Server 2022", shouldFind: true},
		{dirVer: "w11", requested: "Windows Server 2022", shouldFind: true},
		{dirVer: "2k19", requested: "10.0", shouldFind: true},
		{dirVer: "w10", requested: "Windows Server 2019", shouldFind: true},
		{dirVer: "w8.1", requested: "Windows Server 2019", shouldFind: false},
		{dirVer: "w7", requested: "6.1", shouldFind: true},
		{dirVer: "2k8r2", requested: "Windows 7", shouldFind: true},
		{dirVer: "w11", requested: "Windows Server 2019", shouldFind: false},
	}

	for _, tc := range tests {
		got := MatchOSVersion(tc.dirVer, tc.requested)
		if got != tc.shouldFind {
			t.Errorf("MatchOSVersion(%q, %q) = %v, want %v", tc.dirVer, tc.requested, got, tc.shouldFind)
		}
	}
}
