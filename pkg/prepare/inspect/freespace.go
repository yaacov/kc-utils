//go:build unix

package inspect

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

const (
	minBootFreeBytes  int64 = 50 * 1024 * 1024  // 50 MB
	minRootFreeBytes  int64 = 100 * 1024 * 1024 // 100 MB
	minOtherFreeBytes int64 = 10 * 1024 * 1024  // 10 MB
	minFreeInodes     int64 = 100
)

// Record returns free space stats for a mounted path.
func Record(mountRoot string) []types.FreeSpaceInfo {
	var infos []types.FreeSpaceInfo
	for _, rel := range candidateMountRelPaths(mountRoot) {
		info, err := statfsInfo(mountRoot, rel)
		if err != nil {
			slog.Warn("statfs failed", "path", filepath.Join(mountRoot, rel), "error", err)
			continue
		}
		infos = append(infos, info)
	}
	return infos
}

// CheckFreeSpace verifies that mounted guest filesystems have enough free space
// for conversion. Returns an error describing the first mount that fails the
// threshold check, or nil if all mounts have sufficient space.
func CheckFreeSpace(mountRoot string) error {
	mounts := []struct {
		rel       string
		threshold int64
	}{
		{"boot", minBootFreeBytes},
		{"", minRootFreeBytes},
		{filepath.Join("boot", "efi"), minOtherFreeBytes},
	}

	for _, m := range mounts {
		info, err := statfsInfo(mountRoot, m.rel)
		if err != nil {
			continue
		}

		if info.FreeBytes < m.threshold {
			return fmt.Errorf("insufficient free space on %s: %d MB available, need at least %d MB",
				info.Path, info.FreeBytes/(1024*1024), m.threshold/(1024*1024))
		}
		if info.FreeInodes >= 0 && info.FreeInodes < minFreeInodes {
			return fmt.Errorf("insufficient free inodes on %s: %d available, need at least %d",
				info.Path, info.FreeInodes, minFreeInodes)
		}
	}
	return nil
}

func candidateMountRelPaths(mountRoot string) []string {
	candidates := []string{"", "boot", filepath.Join("boot", "efi")}
	var result []string
	for _, rel := range candidates {
		path := filepath.Join(mountRoot, rel)
		if guestio.FileExists(path) {
			result = append(result, rel)
		}
	}
	sort.Slice(result, func(i, j int) bool { return len(result[i]) < len(result[j]) })
	return result
}

func statfsInfo(mountRoot, rel string) (types.FreeSpaceInfo, error) {
	path := filepath.Join(mountRoot, rel)
	if _, err := guestio.FileStat(path); err != nil {
		return types.FreeSpaceInfo{}, err
	}
	freeBytes, freeInodes, err := guestio.FileStatFS(path)
	if err != nil {
		return types.FreeSpaceInfo{}, err
	}

	display := "/" + rel
	if rel == "" {
		display = "/"
	}
	return types.FreeSpaceInfo{
		Path:       display,
		FreeBytes:  freeBytes,
		FreeInodes: freeInodes,
	}, nil
}
