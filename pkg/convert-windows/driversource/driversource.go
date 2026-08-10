package driversource

import "github.com/yaacov/kc-utils/pkg/common/plugin"

type DriverSource interface {
	Available() bool
	FindDrivers(arch, osVersion string, osPrefs, osFallbacks []string) ([]DriverFile, error)
}

type DriverFile struct {
	Name    string
	SrcPath string
	InfPath string
	Arch    string
	// Files are host paths that must be staged for this package (INF/MSI,
	// catalog, and SourceDisksFiles companions). Populated by FilterComplete.
	Files []string
}

var Sources = plugin.NewRegistry[string, DriverSource]()
