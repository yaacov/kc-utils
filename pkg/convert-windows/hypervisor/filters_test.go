package hypervisor

import (
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/registry/mock"
)

func TestRemoveFilter(t *testing.T) {
	h := mock.NewMockHive()
	path := `ControlSet001\Control\Class\{4d36e967-e325-11ce-bfc1-08002be10318}`
	h.CreateKey(path)
	h.SetMultiString(path, "LowerFilters", []string{"partmgr", "prl_strg", "other"})

	RemoveFilter(h, path, "LowerFilters", "prl_strg")

	got, err := h.GetMultiString(path, "LowerFilters")
	if err != nil {
		t.Fatalf("GetMultiString: %v", err)
	}
	for _, v := range got {
		if strings.EqualFold(v, "prl_strg") {
			t.Errorf("prl_strg still present in %v", got)
		}
	}
	if !contains(got, "partmgr") || !contains(got, "other") {
		t.Errorf("kept filters = %v, want partmgr and other", got)
	}
}

func TestRemoveFilterDeletesValueWhenEmpty(t *testing.T) {
	h := mock.NewMockHive()
	path := `ControlSet001\Control\Class\{4d36e97d-e325-11ce-bfc1-08002be10318}`
	h.CreateKey(path)
	h.SetMultiString(path, "UpperFilters", []string{"XENFILT"})

	RemoveFilter(h, path, "UpperFilters", "XENFILT")

	if _, err := h.GetMultiString(path, "UpperFilters"); err == nil {
		t.Fatal("expected UpperFilters value to be deleted when it was the only entry")
	}
}

func TestRemoveFilterMissing(t *testing.T) {
	h := mock.NewMockHive()
	RemoveFilter(h, `missing`, "LowerFilters", "prl_strg")
	if len(h.Ops) > 0 {
		t.Errorf("expected no ops for missing key, got %d", len(h.Ops))
	}
}

func TestDisableService(t *testing.T) {
	h := mock.NewMockHive()
	h.CreateKey(`ControlSet001\Services\prl_strg`)
	DisableService(h, "ControlSet001", "prl_strg")
	start, err := h.GetDWORD(`ControlSet001\Services\prl_strg`, "Start")
	if err != nil {
		t.Fatal(err)
	}
	if start != 4 {
		t.Errorf("Start = %d, want 4", start)
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
