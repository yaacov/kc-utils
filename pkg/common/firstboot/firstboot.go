package firstboot

import "github.com/yaacov/kc-utils/pkg/common/plugin"

type FirstBootHandler interface {
	Install(guestRoot string, commands []string) error
}

var Handlers = plugin.NewRegistry[string, FirstBootHandler]()
