package uefi

import (
	"log/slog"

	"github.com/yaacov/kc-utils/pkg/common/plugin"
	"github.com/yaacov/kc-utils/pkg/common/types"
)

type UEFIEditor interface {
	ConvertToVirtio(guestRoot, espPath string) error
}

var Editors = plugin.NewRegistry[string, UEFIEditor]()

func ConvertAllESPs(mountRoot string, espDevices []string) []types.BlockError {
	var errs []types.BlockError
	for name, editor := range Editors.All() {
		for _, espDev := range espDevices {
			slog.Info("running UEFI editor", "editor", name, "esp", espDev)
			if err := editor.ConvertToVirtio(mountRoot, espDev); err != nil {
				slog.Warn("UEFI editor failed", "editor", name, "esp", espDev, "error", err)
				errs = append(errs, types.BlockError{
					Block: "uefi/" + name, Message: err.Error(),
				})
				continue
			}
			slog.Info("UEFI editor complete", "editor", name, "esp", espDev)
		}
	}
	return errs
}
