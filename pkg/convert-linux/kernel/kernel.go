package kernel

import (
	"github.com/yaacov/kc-utils/pkg/common/plugin"
	"github.com/yaacov/kc-utils/pkg/common/types"
)

// KernelScanner finds installed kernels and selects the best candidate.
type KernelScanner interface {
	ScanKernels(guestRoot string) ([]types.KernelInfo, error)
	SelectBest(kernels []types.KernelInfo) *types.KernelInfo
}

// Scanners is the global registry of KernelScanner implementations.
var Scanners = plugin.NewRegistry[string, KernelScanner]()
