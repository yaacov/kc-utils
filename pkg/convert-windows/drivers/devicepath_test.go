//go:build linux

package drivers

import (
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/registry/mock"
)

func TestUpdateAppendsVirtIO(t *testing.T) {
	hive := mock.NewMockHive()
	key := `Microsoft\Windows\CurrentVersion`
	hive.CreateKey(key)
	hive.SetString(key, "DevicePath", `%SystemRoot%\inf`)

	Update(hive)

	got, err := hive.GetString(key, "DevicePath")
	if err != nil {
		t.Fatalf("GetString: %v", err)
	}
	if !strings.Contains(got, `%SystemRoot%\inf`) {
		t.Errorf("DevicePath missing original: %q", got)
	}
	if !strings.Contains(got, `%SystemRoot%\Drivers\VirtIO`) {
		t.Errorf("DevicePath missing VirtIO: %q", got)
	}
}

func TestUpdateIdempotent(t *testing.T) {
	hive := mock.NewMockHive()
	key := `Microsoft\Windows\CurrentVersion`
	existing := `%SystemRoot%\inf;%SystemRoot%\Drivers\VirtIO`
	hive.CreateKey(key)
	hive.SetExpandString(key, "DevicePath", existing)
	hive.Ops = nil

	Update(hive)

	got, err := hive.GetString(key, "DevicePath")
	if err != nil {
		t.Fatalf("GetString: %v", err)
	}
	if got != existing {
		t.Errorf("DevicePath = %q, want unchanged %q", got, existing)
	}
	if len(hive.Ops) != 0 {
		t.Errorf("expected no writes when VirtIO already present, got %v", hive.Ops)
	}
}

func TestUpdateMissingDefaults(t *testing.T) {
	hive := mock.NewMockHive()

	Update(hive)

	got, err := hive.GetString(`Microsoft\Windows\CurrentVersion`, "DevicePath")
	if err != nil {
		t.Fatalf("GetString: %v", err)
	}
	if !strings.HasPrefix(got, `%SystemRoot%\inf;`) {
		t.Errorf("DevicePath = %q, want default inf prefix", got)
	}
	if !strings.Contains(got, `%SystemRoot%\Drivers\VirtIO`) {
		t.Errorf("DevicePath missing VirtIO: %q", got)
	}
}
