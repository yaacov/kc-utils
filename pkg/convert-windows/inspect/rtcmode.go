package inspect

import (
	"log/slog"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/common/types"
)

// DetectRTCMode reads RealTimeIsUniversal and sets caps.RTCUTC accordingly.
func DetectRTCMode(systemHive registry.Hive, ccs string, caps *types.GuestCaps) {
	caps.RTCUTC = false
	tzPath := ccs + `\Control\TimeZoneInformation`
	rtcVal, rtcErr := systemHive.GetDWORD(tzPath, "RealTimeIsUniversal")
	if rtcErr != nil {
		slog.Warn("RealTimeIsUniversal not found, assuming local time", "error", rtcErr)
	} else if rtcVal == 1 {
		caps.RTCUTC = true
	}
}
