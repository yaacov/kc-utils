//go:build unix

package kudzu

import (
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor"
	"github.com/yaacov/kc-utils/pkg/convert-linux/systemd"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

type Cleanup struct{}

func init() {
	hypervisor.LinuxCleanups.Register("kudzu", &Cleanup{})
}

func (c *Cleanup) Detect(guestRoot string) bool {
	return guestio.FileExists(filepath.Join(guestRoot, "etc", "init.d", "kudzu"))
}

func (c *Cleanup) Cleanup(guestRoot string) error {
	// Equivalent of `chkconfig kudzu off`: remove rc.d symlinks.
	rcDirs, _ := guestio.FileGlob(filepath.Join(guestRoot, "etc", "rc*.d", "[SK]*kudzu"))
	for _, p := range rcDirs {
		_ = guestio.FileRemove(p)
	}
	systemd.DisableSystemdUnit(guestRoot, "kudzu.service")
	return nil
}
