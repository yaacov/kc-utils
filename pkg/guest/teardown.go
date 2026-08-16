//go:build linux

package guest

// TeardownMountRoot best-effort cleans orphaned guest resources under
// mountRoot when prepare-out data is unavailable. Never Syncs.
func TeardownMountRoot(mountRoot string, mode Mode) error {
	f, err := LookupFactory(mode.String())
	if err != nil {
		return err
	}
	return f.TeardownMountRoot(mountRoot)
}
