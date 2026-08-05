package fstrim

import "github.com/yaacov/kc-utils/pkg/common/plugin"

type Trimmer interface {
	Trim(mountpoint string) error
}

var Trimmers = plugin.NewRegistry[string, Trimmer]()
