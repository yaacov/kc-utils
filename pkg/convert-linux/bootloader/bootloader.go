package bootloader

import "github.com/yaacov/kc-utils/pkg/common/plugin"

// BootloaderHandler detects and manages bootloader configuration in the guest.
type BootloaderHandler interface {
	Detect(guestRoot string) bool
	GetDefaultKernel(guestRoot string) (string, error)
	SetDefaultKernel(guestRoot, kernelVersion string) error
	AddKernelArg(guestRoot, arg string) error
	RemoveKernelArg(guestRoot, prefix string) error
	RegenerateConfig(guestRoot string) error
}

// Handlers is the global registry of BootloaderHandler implementations.
var Handlers = plugin.NewRegistry[string, BootloaderHandler]()

// PreferredOrder is the detection order (BLS before grub2).
var PreferredOrder = []string{"bls", "grub2"}

// DetectFirst returns the first matching bootloader in PreferredOrder.
func DetectFirst(guestRoot string) (string, BootloaderHandler) {
	for _, name := range PreferredOrder {
		h, ok := Handlers.Get(name)
		if ok && h.Detect(guestRoot) {
			return name, h
		}
	}
	for name, h := range Handlers.All() {
		if h.Detect(guestRoot) {
			return name, h
		}
	}
	return "", nil
}
