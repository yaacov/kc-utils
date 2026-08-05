package env

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/yaacov/kc-utils/pkg/prepare/guest/overlay"
)

// DiskInfo holds a discovered disk path.
type DiskInfo struct {
	Path string
}

// DiscoverDisks finds attached block devices and disk images.
func DiscoverDisks(cfg *Config) ([]DiskInfo, error) {
	paths, err := globDisks()
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no disks found")
	}
	sort.Slice(paths, func(i, j int) bool {
		return diskNumber(paths[i]) < diskNumber(paths[j])
	})

	disks := make([]DiskInfo, len(paths))
	for i, path := range paths {
		disks[i] = DiskInfo{Path: path}
	}
	slog.Info("discovered conversion disks", "count", len(disks))
	for i, d := range disks {
		kind := "filesystem"
		if strings.HasPrefix(d.Path, "/dev/") {
			kind = "block"
		}
		slog.Info("conversion disk", "index", i, "path", d.Path, "kind", kind)
	}
	return disks, nil
}

func globDisks() ([]string, error) {
	block, err := filepath.Glob(BlockGlob)
	if err != nil {
		return nil, err
	}
	fs, err := filepath.Glob(FSGlob)
	if err != nil {
		return nil, err
	}
	var paths []string
	paths = append(paths, block...)
	for _, p := range fs {
		paths = append(paths, filepath.Join(p, "disk.img"))
	}
	return paths, nil
}

func diskNumber(path string) int {
	re := regexp.MustCompile(`\d+`)
	n, _ := strconv.Atoi(re.FindString(path))
	return n
}

// ToOverlayDisks converts discovered disks for the overlay package.
func ToOverlayDisks(disks []DiskInfo) []*overlay.Disk {
	out := make([]*overlay.Disk, len(disks))
	for i, d := range disks {
		out[i] = &overlay.Disk{BackingPath: d.Path, Path: d.Path}
	}
	return out
}

// ActiveDiskPaths returns the paths currently used by overlay disks.
func ActiveDiskPaths(od []*overlay.Disk) []DiskInfo {
	out := make([]DiskInfo, len(od))
	for i, d := range od {
		out[i] = DiskInfo{Path: d.Path}
	}
	return out
}
