package guestagent

import "github.com/yaacov/kc-utils/pkg/common/plugin"

// FindRequest describes a local package lookup on the conversion host.
type FindRequest struct {
	Name         string
	Format       string // "rpm" or "deb"
	Arch         string // guest arch (e.g. x86_64)
	Distro       string // inspect distro (e.g. rhel, centos)
	MajorVersion int    // inspect major version (e.g. 9)
}

// PackageSource locates local packages on the conversion host
// that can be copied into a guest filesystem for offline installation.
type PackageSource interface {
	Available() bool
	FindPackages(req FindRequest) ([]PackageFile, error)
}

// PackageFile describes a single package file found on the host.
type PackageFile struct {
	Name     string
	FileName string
	HostPath string
	Format   string
	Arch     string
	ELTag    string // e.g. "el9" when found under a versioned tree
}

// Sources is the global registry of PackageSource implementations.
var Sources = plugin.NewRegistry[string, PackageSource]()
