//go:build unix

package parallels

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

func TestDetectViaX86Dir(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "Program Files (x86)", "Parallels", "Parallels Tools")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	u := &Remove{}
	if !u.Detect(root, nil, mock.NewMockHive()) {
		t.Error("Detect = false, want true for x86 Parallels Tools dir")
	}
}

func TestRemove(t *testing.T) {
	root := t.TempDir()
	toolsDir := filepath.Join(root, "Program Files (x86)", "Parallels", "Parallels Tools")
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(toolsDir, "prl_tools.exe"), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	systemHive := mock.NewMockHive()
	systemHive.SetDWORD(`Select`, "Current", 1)
	ccs := "ControlSet001"
	classPath := ccs + `\Control\Class\` + diskClassGUID
	systemHive.CreateKey(classPath)
	systemHive.SetMultiString(classPath, "LowerFilters", []string{"partmgr", "prl_strg"})
	systemHive.CreateKey(ccs + `\Services\prl_strg`)
	systemHive.CreateKey(ccs + `\Services\prl_boot`)

	softwareHive := mock.NewMockHive()
	softwareHive.CreateKey(uninstallKey)

	u := &Remove{}
	if err := u.Remove(root, systemHive, softwareHive); err != nil {
		t.Fatalf("Remove error: %v", err)
	}

	filters, err := systemHive.GetMultiString(classPath, "LowerFilters")
	if err != nil {
		t.Fatalf("GetMultiString: %v", err)
	}
	for _, f := range filters {
		if strings.EqualFold(f, "prl_strg") {
			t.Error("prl_strg still in LowerFilters")
		}
	}
	if !contains(filters, "partmgr") {
		t.Error("partmgr should remain in LowerFilters")
	}

	start, err := systemHive.GetDWORD(ccs+`\Services\prl_strg`, "Start")
	if err != nil {
		t.Fatal(err)
	}
	if start != 4 {
		t.Errorf("prl_strg Start = %d, want 4", start)
	}

	if _, err := os.Stat(toolsDir); !os.IsNotExist(err) {
		t.Error("Parallels Tools directory still exists")
	}
	if softwareHive.KeyExists(uninstallKey) {
		t.Error("uninstall key still exists")
	}
}

func contains(vals []string, want string) bool {
	for _, v := range vals {
		if v == want {
			return true
		}
	}
	return false
}
