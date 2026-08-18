//go:build linux

package guest

import (
	"github.com/yaacov/kc-utils/pkg/backend"
)

// TeardownMountRoot best-effort cleans orphaned guest resources under
// mountRoot when prepare-out data is unavailable. Never Syncs.
func TeardownMountRoot(mountRoot, backendName string) error {
	plugin, err := backend.Lookup(backendName)
	if err != nil {
		return err
	}
	return plugin.TeardownMountRoot(mountRoot)
}
