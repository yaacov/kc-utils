//go:build linux

package kudzu

import (
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type Cleanup struct{}

func init() {
	hypervisor.LinuxCleanups.Register("kudzu", &Cleanup{})
}

func (c *Cleanup) Detect(guestRoot string) bool {
	return guest.FileExists(filepath.Join(guestRoot, "etc", "init.d", "kudzu"))
}

func (c *Cleanup) Cleanup(guestRoot string) error {
	// Equivalent of `chkconfig kudzu off`: remove rc.d symlinks.
	rcDirs, _ := guest.FileGlob(filepath.Join(guestRoot, "etc", "rc*.d", "[SK]*kudzu"))
	for _, p := range rcDirs {
		_ = guest.FileRemove(p)
	}
	hypervisor.DisableSystemdUnit(guestRoot, "kudzu.service")
	return nil
}
