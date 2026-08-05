package driversource

import "github.com/yaacov/kc-utils/pkg/common/plugin"

type DriverSource interface {
	Available() bool
	FindDrivers(arch, osVersion string) ([]DriverFile, error)
}

type DriverFile struct {
	Name    string
	SrcPath string
	InfPath string
	Arch    string
}

var Sources = plugin.NewRegistry[string, DriverSource]()
