//go:build linux

package networkd

import (
	"log/slog"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-linux/network"
	netd "github.com/yaacov/kc-utils/pkg/convert-linux/network/networkd"
)

const handlerName = "networkd"

type Handler struct{}

func init() {
	network.Handlers.Register(handlerName, &Handler{})
}

func (h *Handler) Name() string { return handlerName }

func (h *Handler) Priority() int { return 100 }

func (h *Handler) Detect(guestRoot string) bool {
	return netd.Detect(guestRoot)
}

func (h *Handler) InstallKubeVirtNetworking(guestRoot string) []types.BlockError {
	slog.Info("configuring systemd-networkd for KubeVirt")
	if err := netd.InstallKubeVirtNetworking(guestRoot); err != nil {
		slog.Warn("systemd-networkd KubeVirt networking failed", "error", err)
		return []types.BlockError{{
			Block: "networkd/kubevirt", Message: err.Error(),
		}}
	}
	return nil
}

func (h *Handler) ConfigureStaticIPs(guestRoot string, ips []types.StaticIP) []types.BlockError {
	if len(ips) == 0 {
		return nil
	}
	slog.Debug("configuring static IPs via systemd-networkd")
	if err := netd.WriteStaticNetworks(guestRoot, ips); err != nil {
		slog.Warn("writing systemd-networkd static config failed", "error", err)
		return []types.BlockError{{
			Block: "static-ip/networkd", Message: err.Error(),
		}}
	}
	return nil
}
