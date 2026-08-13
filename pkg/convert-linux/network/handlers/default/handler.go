//go:build linux

package defaulthandler

import (
	"log/slog"

	"github.com/yaacov/kc-utils/pkg/common/firstboot"
	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-linux/network"
	"github.com/yaacov/kc-utils/pkg/convert-linux/network/staticip"
	"github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming"
)

type Handler struct{}

func init() {
	network.Handlers.Register(network.DefaultHandlerName, &Handler{})
}

func (h *Handler) Name() string { return network.DefaultHandlerName }

func (h *Handler) Priority() int { return 0 }

func (h *Handler) Detect(string) bool { return false }

func (h *Handler) InstallKubeVirtNetworking(string) []types.BlockError {
	return nil
}

func (h *Handler) ConfigureStaticIPs(guestRoot string, staticIPs []types.StaticIP) []types.BlockError {
	if len(staticIPs) == 0 {
		return nil
	}

	var errs []types.BlockError

	slog.Debug("preserving NIC naming")
	if err := nicnaming.Apply(guestRoot, staticIPs); err != nil {
		slog.Warn("NIC naming preservation failed", "error", err)
		errs = append(errs, types.BlockError{
			Block: "nic-naming", Message: err.Error(),
		})
	}

	slog.Debug("configuring static IPs")
	if err := staticip.WriteMacToIP(guestRoot, staticIPs); err != nil {
		slog.Warn("writing macToIP failed", "error", err)
		errs = append(errs, types.BlockError{
			Block: "static-ip", Message: err.Error(),
		})
		return errs
	}

	if fbHandler, ok := firstboot.Handlers.Get("systemd"); ok {
		if err := fbHandler.Install(guestRoot, staticip.FirstbootCommands()); err != nil {
			slog.Warn("static IP firstboot install failed", "error", err)
			errs = append(errs, types.BlockError{
				Block: "static-ip/firstboot", Message: err.Error(),
			})
		}
	}

	return errs
}
