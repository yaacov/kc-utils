package suse

import (
	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-linux/distro"
)

type SUSEHandler struct{}

func init() {
	distro.Handlers.Register("suse", &SUSEHandler{})
}

func (s *SUSEHandler) Matches(inspect *types.InspectData) bool {
	return inspect.Distro == "sles" || inspect.Distro == "opensuse-leap" || inspect.Distro == "opensuse-tumbleweed"
}

func (s *SUSEHandler) DefaultKernelArgs() []string {
	return []string{"console=ttyS0"}
}

func (s *SUSEHandler) DefaultConsole() string {
	return "ttyS0"
}
