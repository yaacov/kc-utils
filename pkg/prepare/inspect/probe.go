//go:build unix

package inspect

import (
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

// ProbeRoot checks whether mountPath looks like an OS root filesystem.
// Returns inspect data and true when an OS is detected.
func ProbeRoot(mountPath string) (*types.InspectData, bool) {
	if isWindows(mountPath) {
		data, err := inspectWindows(mountPath)
		if err != nil {
			return nil, false
		}
		return data, true
	}
	if isLinuxRoot(mountPath) {
		data, err := inspectLinux(mountPath)
		if err != nil {
			return nil, false
		}
		return data, true
	}
	return nil, false
}

func isLinuxRoot(root string) bool {
	for _, rel := range LinuxRootMarkerPaths {
		if guestio.FileExists(filepath.Join(root, rel)) {
			return true
		}
	}
	return false
}

// ProductName returns a human-readable OS name from inspect data.
func ProductName(data *types.InspectData) string {
	if data == nil {
		return "unknown"
	}
	if data.ProductName != "" {
		return data.ProductName
	}
	if data.Distro != "" {
		return data.Distro
	}
	return data.Type
}
