package copy

import (
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

var (
	blockGlob = "/dev/block[0-9]*"
	fsGlob    = "/mnt/disks/disk[0-9]*"
)

// SetTargetGlobs overrides target discovery globs for tests. Returns a restore func.
func SetTargetGlobs(block, fs string) func() {
	oldBlock, oldFS := blockGlob, fsGlob
	blockGlob, fsGlob = block, fs
	return func() {
		blockGlob, fsGlob = oldBlock, oldFS
	}
}

// Target describes a PVC write destination.
type Target struct {
	Path       string
	IsBlockDev bool
	Index      int
}

var diskNumRE = regexp.MustCompile(`\d+`)

// DiscoverTargets finds conversion-pod PVC paths (block or filesystem).
func DiscoverTargets() ([]Target, error) {
	block, err := filepath.Glob(blockGlob)
	if err != nil {
		return nil, err
	}
	fsDirs, err := filepath.Glob(fsGlob)
	if err != nil {
		return nil, err
	}

	var targets []Target
	for _, p := range block {
		targets = append(targets, Target{
			Path:       p,
			IsBlockDev: true,
			Index:      diskIndex(p),
		})
	}
	for _, dir := range fsDirs {
		targets = append(targets, Target{
			Path:       filepath.Join(dir, "disk.img"),
			IsBlockDev: false,
			Index:      diskIndex(dir),
		})
	}
	if len(targets) == 0 {
		return nil, nil
	}
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Index < targets[j].Index
	})
	return targets, nil
}

// TargetsFromDir builds file targets {dir}/disk0.img … disk{n-1}.img.
func TargetsFromDir(dir string, n int) []Target {
	if n < 1 {
		return nil
	}
	targets := make([]Target, n)
	for i := 0; i < n; i++ {
		targets[i] = Target{
			Path:       filepath.Join(dir, types.ImageFileName(i)),
			IsBlockDev: false,
			Index:      i,
		}
	}
	return targets
}

func diskIndex(path string) int {
	n, _ := strconv.Atoi(diskNumRE.FindString(path))
	return n
}
