//go:build unix

package hyperv

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/registry/mock"
)

func TestDetectPresent(t *testing.T) {
	hive := mock.NewMockHive()
	ccs := "ControlSet001"
	hive.CreateKey(ccs + `\Services\vmicheartbeat`)

	u := &Services{}
	if !u.Detect("", hive, ccs) {
		t.Error("Detect = false, want true when vmicheartbeat service key exists")
	}
}

func TestDetectAbsent(t *testing.T) {
	hive := mock.NewMockHive()

	u := &Services{}
	if u.Detect("", hive, "ControlSet001") {
		t.Error("Detect = true, want false when no Hyper-V services exist")
	}
}

func TestServiceNames(t *testing.T) {
	u := &Services{}
	names := u.ServiceNames()
	want := map[string]bool{
		"vmicvmsession":   true,
		"storflt":         true,
		"vmickvpexchange": true,
	}
	for _, n := range names {
		delete(want, n)
		if n == "vmicexchange" {
			t.Error("ServiceNames must not include fake name vmicexchange")
		}
	}
	for missing := range want {
		t.Errorf("ServiceNames missing %s", missing)
	}
}

func TestDisableServicesWritesRegistry(t *testing.T) {
	hive := mock.NewMockHive()
	ccs := "ControlSet001"
	hive.CreateKey(ccs + `\Services\vmicheartbeat`)
	hive.CreateKey(ccs + `\Services\storflt`)
	hive.CreateKey(ccs + `\Services\W32Time\TimeProviders\VMICTimeProvider`)
	hive.SetDWORD(ccs+`\Services\W32Time\TimeProviders\VMICTimeProvider`, "Enabled", 1)

	u := &Services{}
	if err := u.DisableServices("", hive, ccs); err != nil {
		t.Fatalf("DisableServices error: %v", err)
	}

	for _, svc := range []string{"vmicheartbeat", "storflt"} {
		start, err := hive.GetDWORD(ccs+`\Services\`+svc, "Start")
		if err != nil {
			t.Fatalf("GetDWORD %s: %v", svc, err)
		}
		if start != 4 {
			t.Errorf("%s Start = %d, want 4 (disabled)", svc, start)
		}
	}

	enabled, err := hive.GetDWORD(ccs+`\Services\W32Time\TimeProviders\VMICTimeProvider`, "Enabled")
	if err != nil {
		t.Fatalf("GetDWORD VMICTimeProvider: %v", err)
	}
	if enabled != 0 {
		t.Errorf("VMICTimeProvider Enabled = %d, want 0", enabled)
	}
}

func TestDisableServicesSkipsMissing(t *testing.T) {
	hive := mock.NewMockHive()

	u := &Services{}
	if err := u.DisableServices("", hive, "ControlSet001"); err != nil {
		t.Fatalf("DisableServices error: %v", err)
	}
	if len(hive.Ops) > 0 {
		t.Errorf("expected no ops for missing services, got %d", len(hive.Ops))
	}
}
