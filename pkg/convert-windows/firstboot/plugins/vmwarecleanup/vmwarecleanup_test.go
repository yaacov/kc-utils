package vmwarecleanup

import (
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"
)

func TestShouldRun(t *testing.T) {
	p := &Plugin{}
	if p.ShouldRun(&firstboot.ContributorConfig{}) {
		t.Error("ShouldRun true when VMwareDriverRemoval unset")
	}
	if !p.ShouldRun(&firstboot.ContributorConfig{
		Options: types.PrepareOptions{VMwareDriverRemoval: true},
	}) {
		t.Error("ShouldRun false when VMwareDriverRemoval set")
	}
}

func TestGenerate(t *testing.T) {
	p := &Plugin{}
	content, err := p.Generate(&firstboot.ContributorConfig{
		Options: types.PrepareOptions{VMwareDriverRemoval: true},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(content, "VMware") {
		t.Errorf("script missing VMware: %q", content)
	}
	if !strings.Contains(content, "sc.exe delete") {
		t.Errorf("script missing sc.exe delete to remove VMware services: %q", content)
	}
	if p.Name() != "cleanup-vmware" {
		t.Errorf("Name = %q", p.Name())
	}
	if p.Priority() != 9100 {
		t.Errorf("Priority = %d", p.Priority())
	}
}
