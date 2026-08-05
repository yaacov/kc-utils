package hypervisor

import "github.com/yaacov/kc-utils/pkg/common/plugin"

// LinuxCleanup removes source hypervisor artifacts from Linux guests.
// Used by kc-convert-linux (pipeline block 11).
type LinuxCleanup interface {
	Detect(guestRoot string) bool
	Cleanup(guestRoot string) error
}

var LinuxCleanups = plugin.NewRegistry[string, LinuxCleanup]()
