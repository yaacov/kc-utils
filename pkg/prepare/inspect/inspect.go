//go:build unix

package inspect

import (
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

// InspectGuest determines OS type and details from a mounted guest root.
func InspectGuest(guestRoot string) (*types.InspectData, error) {
	if isWindows(guestRoot) {
		return inspectWindows(guestRoot)
	}
	return inspectLinux(guestRoot)
}

func isWindows(root string) bool {
	for _, p := range []string{
		filepath.Join(root, "Windows", "System32"),
		filepath.Join(root, "windows", "system32"),
		filepath.Join(root, "WINDOWS", "system32"),
	} {
		if guestio.FileIsDir(p) {
			return true
		}
	}
	return false
}
