package bootconfig

import (
	"log/slog"

	"github.com/yaacov/kc-utils/pkg/convert-linux/bootloader"
	"github.com/yaacov/kc-utils/pkg/convert-linux/distro"
)

// ConfigureConsole updates bootloader console kernel arguments.
func ConfigureConsole(guestRoot string, handler bootloader.BootloaderHandler, distroHandler distro.DistroHandler) {
	if handler == nil {
		return
	}
	consoleDev := "ttyS0"
	if distroHandler != nil {
		consoleDev = distroHandler.DefaultConsole()
	}
	if err := handler.RemoveKernelArg(guestRoot, "rhgb"); err != nil {
		slog.Warn("removing kernel arg failed", "arg", "rhgb", "error", err)
	}
	if err := handler.RemoveKernelArg(guestRoot, "quiet"); err != nil {
		slog.Warn("removing kernel arg failed", "arg", "quiet", "error", err)
	}
	if err := handler.AddKernelArg(guestRoot, "console="+consoleDev); err != nil {
		slog.Warn("adding kernel arg failed", "arg", "console="+consoleDev, "error", err)
	}
}
