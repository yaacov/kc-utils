package bootconfig

import (
	"log/slog"

	"github.com/yaacov/kc-utils/pkg/convert-linux/bootloader"
)

// ConfigureDisplay updates bootloader display/video kernel arguments.
func ConfigureDisplay(guestRoot string, handler bootloader.BootloaderHandler) {
	if handler == nil {
		return
	}
	if err := handler.RemoveKernelArg(guestRoot, "vga"); err != nil {
		slog.Warn("removing kernel arg failed", "arg", "vga", "error", err)
	}
	if err := handler.RemoveKernelArg(guestRoot, "video=cirrus"); err != nil {
		slog.Warn("removing kernel arg failed", "arg", "video=cirrus", "error", err)
	}
	if err := handler.AddKernelArg(guestRoot, "video=virtio"); err != nil {
		slog.Warn("adding kernel arg failed", "arg", "video=virtio", "error", err)
	}
}
