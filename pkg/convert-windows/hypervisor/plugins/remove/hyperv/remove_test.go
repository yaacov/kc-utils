//go:build unix

package hyperv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/registry/mock"
)

func TestDetectUsesCurrentControlSet(t *testing.T) {
	h := mock.NewMockHive()
	h.SetDWORD(`Select`, "Current", 2)
	h.CreateKey(`ControlSet002\Services\vmicheartbeat`)

	u := &Remove{}
	if !u.Detect("", h, nil) {
		t.Error("Detect = false, want true when service exists in active control set")
	}
}

func TestDetectAbsent(t *testing.T) {
	h := mock.NewMockHive()
	h.SetDWORD(`Select`, "Current", 2)

	u := &Remove{}
	if u.Detect("", h, nil) {
		t.Error("Detect = true, want false when no Hyper-V services exist")
	}
}

func TestRemoveIsNoOp(t *testing.T) {
	guestRoot := t.TempDir()
	driversDir := filepath.Join(guestRoot, "Windows", "System32", "drivers")
	if err := os.MkdirAll(driversDir, 0o755); err != nil {
		t.Fatal(err)
	}
	driverPath := filepath.Join(driversDir, "vmbus.sys")
	if err := os.WriteFile(driverPath, []byte("inbox"), 0o644); err != nil {
		t.Fatal(err)
	}

	u := &Remove{}
	if err := u.Remove(guestRoot, mock.NewMockHive(), nil); err != nil {
		t.Fatalf("Remove error: %v", err)
	}
	if _, err := os.Stat(driverPath); err != nil {
		t.Error("inbox Hyper-V driver should remain after Remove")
	}
}
