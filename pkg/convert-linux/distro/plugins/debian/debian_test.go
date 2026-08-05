package debian

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestMatchesDebian(t *testing.T) {
	h := &DebianHandler{}
	inspect := &types.InspectData{Distro: "debian"}
	if !h.Matches(inspect) {
		t.Errorf("expected Matches to return true for distro %q", inspect.Distro)
	}
}

func TestMatchesUbuntu(t *testing.T) {
	h := &DebianHandler{}
	inspect := &types.InspectData{Distro: "ubuntu"}
	if !h.Matches(inspect) {
		t.Errorf("expected Matches to return true for distro %q", inspect.Distro)
	}
}

func TestNoMatchRHEL(t *testing.T) {
	h := &DebianHandler{}
	inspect := &types.InspectData{Distro: "rhel"}
	if h.Matches(inspect) {
		t.Errorf("expected Matches to return false for distro %q", inspect.Distro)
	}
}

func TestDefaultKernelArgs(t *testing.T) {
	h := &DebianHandler{}
	args := h.DefaultKernelArgs()
	if len(args) == 0 {
		t.Fatalf("expected non-empty kernel args, got empty slice")
	}
}

func TestDefaultConsole(t *testing.T) {
	h := &DebianHandler{}
	console := h.DefaultConsole()
	if console != "ttyS0" {
		t.Errorf("expected console %q, got %q", "ttyS0", console)
	}
}
