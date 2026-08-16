//go:build unix

package guest

import "github.com/yaacov/kc-utils/pkg/guest/backend"

// TeardownMountRoot best-effort cleans orphaned guest resources under
// mountRoot when prepare-out data is unavailable. Never Syncs.
func TeardownMountRoot(mountRoot string, mode backend.Mode) error {
	f, err := backend.LookupFactory(mode.String())
	if err != nil {
		return err
	}
	return f.TeardownMountRoot(mountRoot)
}
