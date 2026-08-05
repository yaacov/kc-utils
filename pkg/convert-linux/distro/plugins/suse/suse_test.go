package suse

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestMatchesSLES(t *testing.T) {
	h := &SUSEHandler{}
	inspect := &types.InspectData{Distro: "sles"}
	if !h.Matches(inspect) {
		t.Errorf("expected Matches to return true for distro %q", inspect.Distro)
	}
}

func TestMatchesOpenSUSE(t *testing.T) {
	h := &SUSEHandler{}
	inspect := &types.InspectData{Distro: "opensuse-leap"}
	if !h.Matches(inspect) {
		t.Errorf("expected Matches to return true for distro %q", inspect.Distro)
	}
}

func TestNoMatchRHEL(t *testing.T) {
	h := &SUSEHandler{}
	inspect := &types.InspectData{Distro: "rhel"}
	if h.Matches(inspect) {
		t.Errorf("expected Matches to return false for distro %q", inspect.Distro)
	}
}

func TestDefaultKernelArgs(t *testing.T) {
	h := &SUSEHandler{}
	args := h.DefaultKernelArgs()
	if len(args) == 0 {
		t.Fatalf("expected non-empty kernel args, got empty slice")
	}
}

func TestDefaultConsole(t *testing.T) {
	h := &SUSEHandler{}
	console := h.DefaultConsole()
	if console != "ttyS0" {
		t.Errorf("expected console %q, got %q", "ttyS0", console)
	}
}
