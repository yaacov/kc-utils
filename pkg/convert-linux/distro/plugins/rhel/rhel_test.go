package rhel

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestMatchesRHEL(t *testing.T) {
	h := &RHELHandler{}
	inspect := &types.InspectData{Distro: "rhel"}
	if !h.Matches(inspect) {
		t.Errorf("expected Matches to return true for distro %q", inspect.Distro)
	}
}

func TestMatchesCentOS(t *testing.T) {
	h := &RHELHandler{}
	inspect := &types.InspectData{Distro: "centos"}
	if !h.Matches(inspect) {
		t.Errorf("expected Matches to return true for distro %q", inspect.Distro)
	}
}

func TestMatchesRocky(t *testing.T) {
	h := &RHELHandler{}
	inspect := &types.InspectData{Distro: "rocky"}
	if !h.Matches(inspect) {
		t.Errorf("expected Matches to return true for distro %q", inspect.Distro)
	}
}

func TestMatchesAlmaLinux(t *testing.T) {
	h := &RHELHandler{}
	inspect := &types.InspectData{Distro: "almalinux"}
	if !h.Matches(inspect) {
		t.Errorf("expected Matches to return true for distro %q", inspect.Distro)
	}
}

func TestMatchesFedora(t *testing.T) {
	h := &RHELHandler{}
	inspect := &types.InspectData{Distro: "fedora"}
	if !h.Matches(inspect) {
		t.Errorf("expected Matches to return true for distro %q", inspect.Distro)
	}
}

func TestMatchesAmazonLinux(t *testing.T) {
	h := &RHELHandler{}
	inspect := &types.InspectData{Distro: "amzn"}
	if !h.Matches(inspect) {
		t.Errorf("expected Matches to return true for distro %q", inspect.Distro)
	}
}

func TestNoMatchDebian(t *testing.T) {
	h := &RHELHandler{}
	inspect := &types.InspectData{Distro: "debian"}
	if h.Matches(inspect) {
		t.Errorf("expected Matches to return false for distro %q", inspect.Distro)
	}
}

func TestDefaultKernelArgs(t *testing.T) {
	h := &RHELHandler{}
	args := h.DefaultKernelArgs()
	if len(args) == 0 {
		t.Fatalf("expected non-empty kernel args, got empty slice")
	}
}

func TestDefaultConsole(t *testing.T) {
	h := &RHELHandler{}
	console := h.DefaultConsole()
	if console != "ttyS0" {
		t.Errorf("expected console %q, got %q", "ttyS0", console)
	}
}
