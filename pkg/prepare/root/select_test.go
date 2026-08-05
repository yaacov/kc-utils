package root_test

import (
	"errors"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/prepare/root"
	_ "github.com/yaacov/kc-utils/pkg/prepare/root/plugins/device"
	_ "github.com/yaacov/kc-utils/pkg/prepare/root/plugins/first"
	_ "github.com/yaacov/kc-utils/pkg/prepare/root/plugins/single"
)

func candidates() []types.RootCandidate {
	return []types.RootCandidate{
		{DevicePath: "/dev/loop0p1", ProductName: "RHEL 9"},
		{DevicePath: "/dev/loop0p2", ProductName: "Debian 12"},
	}
}

func TestSelectDefaultEmptyPicksFirst(t *testing.T) {
	got, err := root.Select(candidates(), "")
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if got.DevicePath != "/dev/loop0p1" {
		t.Fatalf("got %q, want first candidate", got.DevicePath)
	}
}

func TestSelectSingleOne(t *testing.T) {
	one := []types.RootCandidate{candidates()[0]}
	got, err := root.Select(one, "single")
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if got.DevicePath != "/dev/loop0p1" {
		t.Fatalf("got %q", got.DevicePath)
	}
}

func TestSelectSingleMultibootFails(t *testing.T) {
	_, err := root.Select(candidates(), "single")
	if err == nil {
		t.Fatal("expected error")
	}
	var mb *root.MultiBootError
	if !errors.As(err, &mb) {
		t.Fatalf("expected MultiBootError, got %T: %v", err, err)
	}
}

func TestSelectFirst(t *testing.T) {
	got, err := root.Select(candidates(), "first")
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if got.DevicePath != "/dev/loop0p1" {
		t.Fatalf("got %q", got.DevicePath)
	}
}

func TestSelectDevice(t *testing.T) {
	got, err := root.Select(candidates(), "/dev/loop0p2")
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if got.DevicePath != "/dev/loop0p2" {
		t.Fatalf("got %q", got.DevicePath)
	}
}

func TestSelectInvalidAsk(t *testing.T) {
	_, err := root.Select(candidates(), "ask")
	if err == nil {
		t.Fatal("expected error")
	}
}
