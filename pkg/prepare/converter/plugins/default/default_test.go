package def

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestSelectLinux(t *testing.T) {
	s := &DefaultSelector{}
	got, err := s.Select(&types.InspectData{Type: "linux"})
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}
	if got != "kc-convert-linux" {
		t.Errorf("Select = %q, want %q", got, "kc-convert-linux")
	}
}

func TestSelectWindows(t *testing.T) {
	s := &DefaultSelector{}
	got, err := s.Select(&types.InspectData{Type: "windows"})
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}
	if got != "kc-convert-windows" {
		t.Errorf("Select = %q, want %q", got, "kc-convert-windows")
	}
}

func TestSelectUnknown(t *testing.T) {
	s := &DefaultSelector{}
	_, err := s.Select(&types.InspectData{Type: "freebsd"})
	if err == nil {
		t.Error("Select should return error for unknown OS type")
	}
}
