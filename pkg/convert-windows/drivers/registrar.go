package drivers

import (
	"github.com/yaacov/kc-utils/pkg/common/plugin"
	"github.com/yaacov/kc-utils/pkg/common/registry"
)

type DriverRegistrar interface {
	Register(hive registry.Hive, ccs string, driverName, driverPath, group, arch string) error
}

var Registrars = plugin.NewRegistry[string, DriverRegistrar]()
