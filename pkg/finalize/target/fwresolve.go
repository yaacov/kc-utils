package target

import "github.com/yaacov/kc-utils/pkg/common/types"

// Target defaults empty firmware to BIOS.
func Target(firmware string) string {
	if firmware == "" {
		return string(types.FirmwareBIOS)
	}
	return firmware
}
