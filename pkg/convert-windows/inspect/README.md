# inspect -- antivirus and RTC detection

Performs pre-conversion inspection of a mounted Windows guest to detect conditions that may affect conversion success or require post-conversion configuration. Currently checks for installed antivirus software and the guest's real-time clock mode.

`DetectAntivirus` enumerates the SOFTWARE hive's Uninstall registry keys and looks for display names containing keywords like "antivirus", "anti-virus", "endpoint protection", or "security center". Matches are returned as warning strings because antivirus software can interfere with VirtIO driver installation and cause INACCESSIBLE_BOOT_DEVICE errors. `DetectRTCMode` reads the `RealTimeIsUniversal` DWORD from the SYSTEM hive's TimeZoneInformation key to determine whether the guest clock is set to UTC or local time, and records the result in `GuestCaps`.

## File layout

| File | Purpose |
|------|---------|
| `antivirus.go` | Scans SOFTWARE hive uninstall keys for antivirus products and returns warning messages |
| `rtcmode.go` | Reads `RealTimeIsUniversal` from the SYSTEM hive and sets `GuestCaps.RTCUTC` accordingly |

## Key exports

| Symbol | Role |
|--------|------|
| `DetectAntivirus` | Scans installed software for antivirus products and returns warning strings |
| `DetectRTCMode` | Reads the RTC mode from the SYSTEM hive and sets `caps.RTCUTC` to true if the clock uses UTC |
