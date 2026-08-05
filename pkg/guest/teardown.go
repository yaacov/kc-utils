//go:build linux

package guest

import (
	"github.com/yaacov/kc-utils/pkg/guest/direct"
	"github.com/yaacov/kc-utils/pkg/guest/guestfs"
)

// TeardownMountRoot best-effort cleans orphaned guest resources under
// mountRoot when prepare-out data is unavailable. Never Syncs.
func TeardownMountRoot(mountRoot string, mode Mode) error {
	switch mode {
	case ModeGuestfs:
		return guestfs.TeardownMountRoot(mountRoot)
	default:
		return direct.TeardownMountRoot(mountRoot)
	}
}
