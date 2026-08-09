package distro

import (
	"log/slog"

	"github.com/yaacov/kc-utils/pkg/common/plugin"
	"github.com/yaacov/kc-utils/pkg/common/types"
)

type DistroHandler interface {
	Matches(inspect *types.InspectData) bool
	DefaultKernelArgs() []string
	DefaultConsole() string
}

var Handlers = plugin.NewRegistry[string, DistroHandler]()

// Format returns the package format (rpm or deb) for a distro ID.
// Only RPM and DEB are supported; unrecognized distros default to RPM.
func Format(distro string) string {
	switch distro {
	case "debian", "ubuntu":
		return "deb"
	case "rhel", "centos", "rocky", "almalinux", "alma", "ol", "fedora", "amzn",
		"sles", "opensuse-leap", "opensuse-tumbleweed",
		"alt":
		return "rpm"
	default:
		slog.Warn("unrecognized distro, defaulting to rpm package format", "distro", distro)
		return "rpm"
	}
}

// Name returns the package manager command family for a distro ID.
func Name(distro string) string {
	switch distro {
	case "debian", "ubuntu", "alt":
		return "apt"
	case "sles", "opensuse-leap", "opensuse-tumbleweed":
		return "zypper"
	default:
		return "dnf"
	}
}
