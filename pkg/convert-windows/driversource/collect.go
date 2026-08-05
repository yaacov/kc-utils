package driversource

import (
	"fmt"
	"log/slog"
)

// Cleaner is implemented by driver sources that hold temporary resources
// (for example an extracted virtio-win ISO tree) that must outlive FindDrivers
// until after drivers are copied into the guest.
type Cleaner interface {
	Cleanup()
}

// CollectDrivers finds virtio-win drivers from the ISO source only.
// Call Cleanup on any returned Cleaner values after drivers.Copy completes.
func CollectDrivers(arch, osVersion string) ([]DriverFile, []Cleaner, error) {
	src, ok := Sources.Get("iso")
	if !ok {
		return nil, nil, fmt.Errorf("iso driver source not registered")
	}
	if !src.Available() {
		return nil, nil, fmt.Errorf("virtio-win ISO not available (expected /usr/share/virtio-win/virtio-win.iso)")
	}

	files, err := src.FindDrivers(arch, osVersion)
	if err != nil {
		return nil, nil, fmt.Errorf("iso driver source: %w", err)
	}
	if len(files) == 0 {
		return nil, nil, fmt.Errorf("no virtio-win drivers found in ISO for arch=%s os=%s", arch, osVersion)
	}

	slog.Info("found drivers", "count", len(files), "source", "iso")
	var cleaners []Cleaner
	if c, ok := src.(Cleaner); ok {
		cleaners = append(cleaners, c)
	}
	return files, cleaners, nil
}

// CleanupAll runs Cleanup on each cleaner, ignoring individual failures.
func CleanupAll(cleaners []Cleaner) {
	for _, c := range cleaners {
		if c != nil {
			c.Cleanup()
		}
	}
}
