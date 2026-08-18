//go:build unix

package deb

import (
	"path/filepath"
	"sort"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-linux/kernel"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type Scanner struct{}

func init() {
	kernel.Scanners.Register("deb", &Scanner{})
}

func (s *Scanner) ScanKernels(guestRoot string) ([]types.KernelInfo, error) {
	modulesDir := kernel.ModulesDir(guestRoot)
	entries, err := guest.FileReadDir(modulesDir)
	if err != nil {
		return nil, err
	}

	bootFiles := readBootNames(guestRoot)

	var kernels []types.KernelInfo
	for _, e := range entries {
		if !e.IsDir {
			continue
		}
		ver := e.Name
		vmlinuz := findVmlinuz(bootFiles, ver)
		initrd := findInitrd(bootFiles, ver)
		var hasVirtio, isXenPV bool
		if vmlinuz != "" {
			hasVirtio, isXenPV = kernel.ProbeModules(guestRoot, ver)
		}
		kernels = append(kernels, types.KernelInfo{
			Version:    ver,
			Path:       vmlinuz,
			InitrdPath: initrd,
			HasVirtio:  hasVirtio,
			IsXenPV:    isXenPV,
		})
	}

	sort.Slice(kernels, func(i, j int) bool {
		return kernels[i].Version > kernels[j].Version
	})
	return kernels, nil
}

// readBootNames reads /boot once and returns a set of filenames.
func readBootNames(guestRoot string) map[string]bool {
	entries, err := guest.FileReadDir(filepath.Join(guestRoot, "boot"))
	if err != nil {
		return nil
	}
	names := make(map[string]bool, len(entries))
	for _, e := range entries {
		names[e.Name] = true
	}
	return names
}

func findVmlinuz(bootFiles map[string]bool, ver string) string {
	for _, name := range []string{"vmlinuz-" + ver, "vmlinux-" + ver} {
		if bootFiles[name] {
			return "/boot/" + name
		}
	}
	return ""
}

func findInitrd(bootFiles map[string]bool, ver string) string {
	for _, name := range []string{"initrd.img-" + ver, "initramfs-" + ver + ".img"} {
		if bootFiles[name] {
			return "/boot/" + name
		}
	}
	return ""
}

func (s *Scanner) SelectBest(kernels []types.KernelInfo) *types.KernelInfo {
	for i := range kernels {
		if kernels[i].Path != "" {
			return &kernels[i]
		}
	}
	if len(kernels) > 0 {
		return &kernels[0]
	}
	return nil
}
