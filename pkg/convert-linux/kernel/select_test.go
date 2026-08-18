//go:build unix

package kernel

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestBestPreferVirtio(t *testing.T) {
	kernels := []types.KernelInfo{
		{Version: "5.14.0-100", Path: "/boot/vmlinuz-5.14.0-100", HasVirtio: false},
		{Version: "5.14.0-99", Path: "/boot/vmlinuz-5.14.0-99", HasVirtio: true},
	}
	best := Best(kernels)
	if best == nil || !best.HasVirtio {
		t.Fatalf("expected virtio kernel, got %v", best)
	}
}

func TestBestFilterXenPV(t *testing.T) {
	kernels := []types.KernelInfo{
		{Version: "5.14.0-100", Path: "/boot/vmlinuz-5.14.0-100", IsXenPV: true, HasVirtio: true},
		{Version: "5.14.0-50", Path: "/boot/vmlinuz-5.14.0-50", HasVirtio: false},
	}
	best := Best(kernels)
	if best == nil || best.IsXenPV || best.Version != "5.14.0-50" {
		t.Fatalf("unexpected kernel: %v", best)
	}
}

func TestBestVersionOrdering(t *testing.T) {
	kernels := []types.KernelInfo{
		{Version: "5.14.0-100", Path: "/boot/vmlinuz-5.14.0-100", HasVirtio: true},
		{Version: "5.14.0-200", Path: "/boot/vmlinuz-5.14.0-200", HasVirtio: true},
	}
	best := Best(kernels)
	if best == nil || best.Version != "5.14.0-200" {
		t.Fatalf("expected 5.14.0-200, got %v", best)
	}
}

func TestBestSkipsUnbootable(t *testing.T) {
	kernels := []types.KernelInfo{
		{Version: "5.14.0-421", Path: ""},
		{Version: "5.14.0-200", Path: "/boot/vmlinuz-5.14.0-200", HasVirtio: true},
	}
	best := Best(kernels)
	if best == nil || best.Version != "5.14.0-200" {
		t.Fatalf("expected bootable kernel 5.14.0-200, got %v", best)
	}
}

func TestBestAllUnbootable(t *testing.T) {
	kernels := []types.KernelInfo{
		{Version: "5.14.0-421", Path: ""},
		{Version: "5.14.0-200", Path: ""},
	}
	best := Best(kernels)
	if best != nil {
		t.Fatalf("expected nil when all kernels are unbootable, got %v", best)
	}
}

func TestBestEmpty(t *testing.T) {
	if Best(nil) != nil {
		t.Fatal("expected nil for empty slice")
	}
}
