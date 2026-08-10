//go:build linux

package citrix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/registry/mock"
)

func TestDetectPresent(t *testing.T) {
	h := mock.NewMockHive()
	h.CreateKey(uninstallKey)

	u := &Remove{}
	if !u.Detect("/fake", nil, h) {
		t.Error("Detect = false, want true when uninstall key exists")
	}
}

func TestRemove(t *testing.T) {
	root := t.TempDir()
	toolsDir := filepath.Join(root, "Program Files", "Citrix", "XenTools")
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(toolsDir, "xe.exe"), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	systemHive := mock.NewMockHive()
	systemHive.SetDWORD(`Select`, "Current", 1)
	ccs := "ControlSet001"
	for _, guid := range []string{systemClassGUID, hdcClassGUID} {
		classPath := ccs + `\Control\Class\` + guid
		systemHive.CreateKey(classPath)
		systemHive.SetMultiString(classPath, "UpperFilters", []string{"PartMgr", "XENFILT"})
	}
	systemHive.CreateKey(ccs + `\Services\XenSvc`)

	softwareHive := mock.NewMockHive()
	softwareHive.CreateKey(uninstallKey)

	u := &Remove{}
	if err := u.Remove(root, systemHive, softwareHive); err != nil {
		t.Fatalf("Remove error: %v", err)
	}

	for _, guid := range []string{systemClassGUID, hdcClassGUID} {
		classPath := ccs + `\Control\Class\` + guid
		filters, err := systemHive.GetMultiString(classPath, "UpperFilters")
		if err != nil {
			t.Fatalf("GetMultiString %s: %v", guid, err)
		}
		for _, f := range filters {
			if strings.EqualFold(f, "XENFILT") {
				t.Errorf("XENFILT still in UpperFilters for %s", guid)
			}
		}
	}

	start, err := systemHive.GetDWORD(ccs+`\Services\XenSvc`, "Start")
	if err != nil {
		t.Fatal(err)
	}
	if start != 4 {
		t.Errorf("XenSvc Start = %d, want 4", start)
	}

	if _, err := os.Stat(toolsDir); !os.IsNotExist(err) {
		t.Error("Citrix XenTools directory still exists")
	}
	if softwareHive.KeyExists(uninstallKey) {
		t.Error("uninstall key still exists")
	}
}
