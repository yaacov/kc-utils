//go:build unix

package mount

import (
	"github.com/yaacov/kc-utils/pkg/common/plugin"
	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest"
)

// MountSpec describes one guest filesystem mount.
type MountSpec struct {
	DevicePath string
	GuestMP    string
	FSType     string
	ReadOnly   bool
}

// PlanContext carries state for mount planning.
type PlanContext struct {
	MountRoot  string
	Root       types.RootCandidate
	Disks      []types.DiskInfo
	Firmware   types.FirmwareInfo
	LVPaths    []string
	AllDevices []string
	Guest      *guest.Guest
}

// DetectFS returns the filesystem type via the mode-aware guest wrapper.
func (ctx *PlanContext) DetectFS(device string) (string, error) {
	if ctx.Guest != nil {
		return ctx.Guest.FSType(device)
	}
	return "", nil
}

// MountPlanner builds mount plans for a chosen root.
type MountPlanner interface {
	Matches(inspect *types.InspectData) bool
	Plan(ctx *PlanContext) ([]MountSpec, error)
	Expand(ctx *PlanContext, guestRootHost string) ([]MountSpec, error)
}

// Planners is the global registry of MountPlanner implementations.
var Planners = plugin.NewRegistry[string, MountPlanner]()

// Plan selects a planner and returns its registry name plus initial mount specs.
func Plan(ctx *PlanContext) (string, MountPlanner, []MountSpec, error) {
	for name, planner := range Planners.All() {
		if planner.Matches(&ctx.Root.Inspect) {
			specs, err := planner.Plan(ctx)
			return name, planner, specs, err
		}
	}
	return "", nil, nil, nil
}
