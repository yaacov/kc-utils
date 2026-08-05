package remap

import "github.com/yaacov/kc-utils/pkg/common/plugin"

// DeviceRemapper rewrites block device names in guest configuration files.
type DeviceRemapper interface {
	Name() string
	Detect(guestRoot string) bool
	Remap(guestRoot string) error
}

var Remappers = plugin.NewRegistry[string, DeviceRemapper]()
