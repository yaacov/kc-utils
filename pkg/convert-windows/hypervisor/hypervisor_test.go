package hypervisor

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/registry/mock"
)

func TestCurrentControlSetFromHive(t *testing.T) {
	h := mock.NewMockHive()
	h.SetDWORD(`Select`, "Current", 2)

	if got := CurrentControlSet(h); got != "ControlSet002" {
		t.Errorf("CurrentControlSet = %q, want ControlSet002", got)
	}
}

func TestCurrentControlSetFallback(t *testing.T) {
	h := mock.NewMockHive()

	if got := CurrentControlSet(h); got != "ControlSet001" {
		t.Errorf("CurrentControlSet = %q, want ControlSet001", got)
	}
}
