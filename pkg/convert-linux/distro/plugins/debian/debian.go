package debian

import (
	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-linux/distro"
)

type DebianHandler struct{}

func init() {
	distro.Handlers.Register("debian", &DebianHandler{})
}

func (d *DebianHandler) Matches(inspect *types.InspectData) bool {
	return inspect.Distro == "debian" || inspect.Distro == "ubuntu"
}

func (d *DebianHandler) DefaultKernelArgs() []string {
	return []string{"console=ttyS0"}
}

func (d *DebianHandler) DefaultConsole() string {
	return "ttyS0"
}
