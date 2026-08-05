package signal

import (
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"
)

func TestShouldRun(t *testing.T) {
	p := &Plugin{}
	if p.ShouldRun(&firstboot.ContributorConfig{}) {
		t.Error("ShouldRun true when WaitForGuestReboot unset")
	}
	if !p.ShouldRun(&firstboot.ContributorConfig{
		Options: types.PrepareOptions{WaitForGuestReboot: true},
	}) {
		t.Error("ShouldRun false when WaitForGuestReboot set")
	}
}

func TestGenerate(t *testing.T) {
	p := &Plugin{}
	content, err := p.Generate(&firstboot.ContributorConfig{
		Options: types.PrepareOptions{WaitForGuestReboot: true},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(content, "CONVERSION_DONE") {
		t.Errorf("script missing CONVERSION_DONE: %q", content)
	}
	if p.Name() != "signal-conversion-done" {
		t.Errorf("Name = %q", p.Name())
	}
	if p.Priority() != 99999 {
		t.Errorf("Priority = %d", p.Priority())
	}
}
