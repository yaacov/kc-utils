package inspect

import (
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/registry/mock"
	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestDetectAntivirus(t *testing.T) {
	hive := mock.NewMockHive()
	base := `Microsoft\Windows\CurrentVersion\Uninstall`

	avKey := base + `\AVProduct`
	hive.CreateKey(avKey)
	hive.SetString(avKey, "DisplayName", "Acme Antivirus Pro")

	otherKey := base + `\Notepad`
	hive.CreateKey(otherKey)
	hive.SetString(otherKey, "DisplayName", "Notepad++")

	epKey := base + `\Endpoint`
	hive.CreateKey(epKey)
	hive.SetString(epKey, "DisplayName", "Corp Endpoint Protection")

	warnings := DetectAntivirus(hive)
	if len(warnings) != 2 {
		t.Fatalf("warnings = %v, want 2", warnings)
	}
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, "Acme Antivirus Pro") {
		t.Errorf("missing antivirus warning: %v", warnings)
	}
	if !strings.Contains(joined, "Corp Endpoint Protection") {
		t.Errorf("missing endpoint protection warning: %v", warnings)
	}
	if strings.Contains(joined, "Notepad") {
		t.Errorf("non-AV product should not warn: %v", warnings)
	}
}

func TestDetectAntivirusEmpty(t *testing.T) {
	hive := mock.NewMockHive()
	warnings := DetectAntivirus(hive)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
}

func TestDetectRTCModeUTC(t *testing.T) {
	hive := mock.NewMockHive()
	ccs := "ControlSet001"
	tzPath := ccs + `\Control\TimeZoneInformation`
	hive.CreateKey(tzPath)
	hive.SetDWORD(tzPath, "RealTimeIsUniversal", 1)

	caps := &types.GuestCaps{}
	DetectRTCMode(hive, ccs, caps)
	if !caps.RTCUTC {
		t.Error("RTCUTC = false, want true")
	}
}

func TestDetectRTCModeLocal(t *testing.T) {
	hive := mock.NewMockHive()
	ccs := "ControlSet001"
	tzPath := ccs + `\Control\TimeZoneInformation`
	hive.CreateKey(tzPath)
	hive.SetDWORD(tzPath, "RealTimeIsUniversal", 0)

	caps := &types.GuestCaps{RTCUTC: true}
	DetectRTCMode(hive, ccs, caps)
	if caps.RTCUTC {
		t.Error("RTCUTC = true, want false when DWORD is 0")
	}
}

func TestDetectRTCModeMissing(t *testing.T) {
	hive := mock.NewMockHive()
	caps := &types.GuestCaps{RTCUTC: true}
	DetectRTCMode(hive, "ControlSet001", caps)
	if caps.RTCUTC {
		t.Error("RTCUTC = true, want false when key missing")
	}
}
