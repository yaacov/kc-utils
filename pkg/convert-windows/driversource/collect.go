package driversource

import "fmt"

// CollectDrivers finds virtio-win drivers from the pre-extracted directory tree.
func CollectDrivers(arch, osVersion string) ([]DriverFile, error) {
	src, ok := Sources.Get("directory")
	if !ok || !src.Available() {
		return nil, fmt.Errorf("virtio-win driver tree not available (expected /usr/share/virtio-win/drivers/by-os)")
	}

	files, err := src.FindDrivers(arch, osVersion)
	if err != nil {
		return nil, fmt.Errorf("directory driver source: %w", err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no virtio-win drivers found for arch=%s os=%s", arch, osVersion)
	}
	return files, nil
}
