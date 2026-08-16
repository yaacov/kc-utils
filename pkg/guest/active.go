//go:build unix

package guest

import (
	"fmt"
	"sync"
)

var (
	activeMu sync.RWMutex
	active   *Guest
)

// SetActive registers the current guest handle for packages that cannot
// take an explicit *Guest (bootloader plugins, remappers, customizers).
func SetActive(g *Guest) {
	activeMu.Lock()
	defer activeMu.Unlock()
	active = g
}

// ClearActive clears the process-wide guest handle.
func ClearActive() {
	SetActive(nil)
}

// Active returns the process-wide guest handle, or nil.
func Active() *Guest {
	activeMu.RLock()
	defer activeMu.RUnlock()
	return active
}

// RunInGuest runs cmd in the guest via the active handle.
func RunInGuest(guestRoot string, cmd []string) ([]byte, error) {
	g := Active()
	if g == nil {
		return nil, fmt.Errorf("no active guest handle")
	}
	return g.RunCommand(guestRoot, cmd)
}

// BlkidUUID returns the UUID for device via the active handle.
func BlkidUUID(device string) string {
	g := Active()
	if g == nil {
		return ""
	}
	v, err := g.BlkidAttr(device, "UUID")
	if err != nil {
		return ""
	}
	return v
}
