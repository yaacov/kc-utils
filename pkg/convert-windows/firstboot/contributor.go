package firstboot

import (
	"github.com/yaacov/kc-utils/pkg/common/plugin"
	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-windows/driversource"
)

// ContributorConfig holds the context passed to each firstboot contributor.
type ContributorConfig struct {
	MountRoot   string
	Offline     bool
	DriverFiles []driversource.DriverFile
	StaticIPs   []types.StaticIP
	Options     types.PrepareOptions
}

// Contributor generates a single firstboot PowerShell script.
type Contributor interface {
	// Priority determines script execution order (lower runs first).
	Priority() int
	// ShouldRun returns whether this contributor should generate a script.
	ShouldRun(cfg *ContributorConfig) bool
	// Generate returns the PowerShell script content.
	Generate(cfg *ContributorConfig) (string, error)
	// Name returns a short identifier used in the script filename.
	Name() string
}

// Contributors is the global registry for firstboot script contributors.
var Contributors = plugin.NewRegistry[string, Contributor]()
