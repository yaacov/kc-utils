package guestagent

import "github.com/yaacov/kc-utils/pkg/common/plugin"

type GuestAgent interface {
	Detect(guestRoot string) bool
	Remove(guestRoot string) error
}

var Agents = plugin.NewRegistry[string, GuestAgent]()
