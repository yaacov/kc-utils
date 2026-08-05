package kernel

import "github.com/yaacov/kc-utils/pkg/common/types"

// Best filters out Xen PV-only and unbootable kernels, then selects the
// best candidate. A kernel without a vmlinuz (Path) is considered unbootable
// (e.g. a leftover /lib/modules directory from an uninstalled package).
func Best(kernels []types.KernelInfo) *types.KernelInfo {
	var best *types.KernelInfo
	for i := range kernels {
		k := &kernels[i]
		if k.IsXenPV {
			continue
		}
		if k.Path == "" {
			continue
		}
		if best == nil {
			best = k
			continue
		}
		if k.HasVirtio && !best.HasVirtio {
			best = k
			continue
		}
		if k.Version > best.Version {
			best = k
		}
	}
	return best
}
