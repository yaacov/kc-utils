package customize

import "github.com/yaacov/kc-utils/pkg/common/plugin"

type Customizer interface {
	Apply(guestRoot string, options map[string]string) error
}

var Customizers = plugin.NewRegistry[string, Customizer]()
