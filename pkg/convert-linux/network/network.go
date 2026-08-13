//go:build linux

// Package network selects an exclusive guest network handler for pipeline blocks
// 11b and 15 based on the guest's active network stack.
package network

import (
	"log/slog"

	"github.com/yaacov/kc-utils/pkg/common/plugin"
	"github.com/yaacov/kc-utils/pkg/common/types"
)

const DefaultHandlerName = "default"

// Handler configures guest networking during offline conversion. Exactly one
// handler is selected per guest via Select.
type Handler interface {
	// Name returns the handler key for logging and BlockError labels.
	Name() string
	// Priority breaks ties when multiple handlers Detect true. Higher wins.
	Priority() int
	// Detect reports whether this handler manages the guest's active network stack.
	// Must use service-state signals, not artifact presence alone.
	Detect(guestRoot string) bool
	// InstallKubeVirtNetworking runs in pipeline block 11b (may be no-op).
	InstallKubeVirtNetworking(guestRoot string) []types.BlockError
	// ConfigureStaticIPs runs in pipeline block 15.
	ConfigureStaticIPs(guestRoot string, ips []types.StaticIP) []types.BlockError
}

// Handlers is the global registry for guest network stack handlers.
var Handlers = plugin.NewRegistry[string, Handler]()

// Select returns exactly one handler for the guest. Non-default handlers are
// chosen by Detect and Priority; when none match, the default handler is used.
func Select(guestRoot string) Handler {
	var best Handler
	bestPriority := -1
	for name, h := range Handlers.All() {
		if name == DefaultHandlerName {
			continue
		}
		if !h.Detect(guestRoot) {
			continue
		}
		if h.Priority() > bestPriority {
			best = h
			bestPriority = h.Priority()
		}
	}
	if best != nil {
		return best
	}
	if h, ok := Handlers.Get(DefaultHandlerName); ok {
		return h
	}
	slog.Error("no network handlers registered")
	return nil
}

// PrimaryLabel returns a stable axis label for the selected handler.
func PrimaryLabel(h Handler) string {
	if h != nil && h.Name() == "networkd" {
		return types.NetworkPrimarySystemdNetworkd
	}
	return types.NetworkPrimaryLegacy
}
