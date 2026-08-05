package version_test

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/convert-windows/version"
)

func TestCollectGuestAgentMSI(t *testing.T) {
	tests := []struct {
		handler string
		want    bool
	}{
		{"win10", true},
		{"win2008r2", true},
		{"win2008", false},
		{"winvista", false},
		{"win2003", false},
		{"winxp", false},
	}
	for _, tc := range tests {
		if got := version.CollectGuestAgentMSI(tc.handler); got != tc.want {
			t.Errorf("CollectGuestAgentMSI(%q) = %v, want %v", tc.handler, got, tc.want)
		}
	}
}
