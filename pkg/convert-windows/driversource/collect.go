package driversource

import (
	"fmt"
	"strings"

	"github.com/yaacov/kc-utils/pkg/convert-windows/version"
)

const preWin8VendorHint = "pre–Win 8 virtio-win dirs (2k8, 2k3, xp, vista) require virtio-win 1.9.12-4.el7 staged at image build; see build/kc-v2v/vendor/README.md"

var preWin8OSDirs = map[string]struct{}{
	"2k8":   {},
	"2k3":   {},
	"xp":    {},
	"vista": {},
}

// CollectDrivers finds virtio-win drivers from the pre-extracted directory tree.
func CollectDrivers(arch, osVersion, handlerName string, osPrefs, osFallbacks []string) ([]DriverFile, error) {
	src, ok := Sources.Get("directory")
	if !ok || !src.Available() {
		return nil, fmt.Errorf("virtio-win driver tree not available (expected /usr/share/virtio-win/drivers/by-os)")
	}

	files, err := src.FindDrivers(arch, osVersion, osPrefs, osFallbacks)
	if err != nil {
		return nil, enrichDriverLookupError(handlerName, osVersion, osPrefs,
			fmt.Errorf("directory driver source: %w", err))
	}
	if len(files) == 0 {
		return nil, enrichDriverLookupError(handlerName, osVersion, osPrefs,
			fmt.Errorf("no virtio-win drivers found for arch=%s os=%s", arch, osVersion))
	}
	if !version.CollectGuestAgentMSI(handlerName) {
		files = omitGuestAgentFiles(files)
	}
	return files, nil
}

func omitGuestAgentFiles(files []DriverFile) []DriverFile {
	filtered := make([]DriverFile, 0, len(files))
	for _, f := range files {
		if f.Name == "qemu-ga" {
			continue
		}
		filtered = append(filtered, f)
	}
	return filtered
}

func enrichDriverLookupError(handlerName, osVersion string, osPrefs []string, err error) error {
	if err == nil || len(osPrefs) == 0 {
		return err
	}
	required := osPrefs[0]
	msg := fmt.Sprintf("handler=%s guest=%q required dir=%q: %v", handlerName, osVersion, required, err)
	if _, ok := preWin8OSDirs[strings.ToLower(required)]; ok {
		msg += "; " + preWin8VendorHint
	}
	return fmt.Errorf("%s", msg)
}
