package rhel

import (
	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-linux/distro"
)

type RHELHandler struct{}

func init() {
	distro.Handlers.Register("rhel", &RHELHandler{})
}

func (r *RHELHandler) Matches(inspect *types.InspectData) bool {
	switch inspect.Distro {
	case "rhel", "centos", "rocky", "almalinux", "ol", "fedora", "amzn":
		return true
	default:
		return false
	}
}

func (r *RHELHandler) DefaultKernelArgs() []string {
	return []string{"console=ttyS0", "crashkernel=auto"}
}

func (r *RHELHandler) DefaultConsole() string {
	return "ttyS0"
}
