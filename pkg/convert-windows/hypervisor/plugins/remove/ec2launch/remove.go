//go:build unix

package ec2launch

import (
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

var uninstallKeys = []string{
	`Microsoft\Windows\CurrentVersion\Uninstall\EC2Launch`,
	`Microsoft\Windows\CurrentVersion\Uninstall\Ec2Config`,
	`Microsoft\Windows\CurrentVersion\Uninstall\Amazon EC2Launch`,
}

type Remove struct{}

func init() {
	hypervisor.WindowsRemoves.Register("ec2launch", &Remove{})
}

func (r *Remove) Detect(guestRoot string, _, softwareHive registry.Hive) bool {
	for _, key := range uninstallKeys {
		if softwareHive.KeyExists(key) {
			return true
		}
	}
	dirs := []string{
		filepath.Join(guestRoot, "Program Files", "Amazon", "EC2Launch"),
		filepath.Join(guestRoot, "Program Files", "Amazon", "Ec2ConfigService"),
	}
	for _, d := range dirs {
		if guestio.FileExists(d) {
			return true
		}
	}
	return false
}

func (r *Remove) Remove(guestRoot string, _, softwareHive registry.Hive) error {
	for _, key := range uninstallKeys {
		if softwareHive.KeyExists(key) {
			softwareHive.DeleteKey(key)
		}
	}

	removePaths := []string{
		filepath.Join(guestRoot, "Program Files", "Amazon", "EC2Launch"),
		filepath.Join(guestRoot, "Program Files", "Amazon", "Ec2ConfigService"),
	}
	for _, p := range removePaths {
		_ = guestio.FileRemoveAll(p)
	}

	return nil
}
