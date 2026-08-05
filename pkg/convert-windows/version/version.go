package version

import (
	"log/slog"

	"github.com/yaacov/kc-utils/pkg/common/plugin"
	"github.com/yaacov/kc-utils/pkg/common/types"
)

// LauncherKind selects the firstboot.bat template.
type LauncherKind int

const (
	LauncherModern LauncherKind = iota
	LauncherPSV1
	LauncherBatOnly
)

// StaticIPKind selects static IP firstboot script generation.
type StaticIPKind int

const (
	StaticIPNetCmdlet StaticIPKind = iota
	StaticIPRegistry
	StaticIPWMINetsh
)

// DiskOnlineKind selects disk online firstboot behavior.
type DiskOnlineKind int

const (
	DiskOnlineGetDisk DiskOnlineKind = iota
	DiskOnlineWMIDiskpart
	DiskOnlineSkip
)

// VMwareCleanupKind selects VMware cleanup firstboot behavior.
type VMwareCleanupKind int

const (
	VMwareCleanupPNP VMwareCleanupKind = iota
	VMwareCleanupDevconBat
	VMwareCleanupSkip
)

// VersionHandler classifies a Windows guest and drives version-specific conversion behavior.
type VersionHandler interface {
	Name() string
	Matches(inspect *types.InspectData) bool

	DriverOSPreferences() []string
	DriverOSFallbacks() []string

	FirstbootLauncher() LauncherKind
	SupportsPowerShell() bool
	StaticIPMode() StaticIPKind
	DiskOnlineMode() DiskOnlineKind
	VMwareCleanupMode() VMwareCleanupKind
}

// Handlers is the global registry of Windows version handlers.
var Handlers = plugin.NewRegistry[string, VersionHandler]()

var classifyOrder []string

// Register adds a handler and records classification order (most specific first).
func Register(name string, handler VersionHandler) {
	Handlers.Register(name, handler)
	classifyOrder = append(classifyOrder, name)
}

// Classify returns the first matching version handler, or winunknown.
func Classify(inspect *types.InspectData) VersionHandler {
	if inspect == nil {
		return handlerUnknown
	}
	for _, name := range classifyOrder {
		h, ok := Handlers.Get(name)
		if !ok {
			continue
		}
		if h.Matches(inspect) {
			slog.Debug("matched version handler", "name", name)
			return h
		}
	}
	slog.Warn("no version handler matched, using winunknown")
	return handlerUnknown
}
