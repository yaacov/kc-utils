//go:build linux

package virtualbox

import (
	"bufio"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type Cleanup struct{}

func init() {
	hypervisor.LinuxCleanups.Register("virtualbox", &Cleanup{})
}

func (c *Cleanup) Detect(guestRoot string) bool {
	return guest.FileExists(filepath.Join(guestRoot, "usr", "bin", "VBoxService")) ||
		guest.FileExists(filepath.Join(guestRoot, "usr", "sbin", "VBoxService")) ||
		guest.FileExists(filepath.Join(guestRoot, "var", "lib", "VBoxGuestAdditions", "config"))
}

func (c *Cleanup) Cleanup(guestRoot string) error {
	for _, unit := range []string{"vboxadd-service.service", "vboxadd.service", "vboxservice.service"} {
		hypervisor.DisableSystemdUnit(guestRoot, unit)
	}

	cfgPath := filepath.Join(guestRoot, "var", "lib", "VBoxGuestAdditions", "config")
	if data, err := guest.FileRead(cfgPath); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(line, "INSTALL_DIR=") {
				continue
			}
			dir := strings.Trim(strings.TrimPrefix(line, "INSTALL_DIR="), `"'`)
			if strings.HasPrefix(dir, "/") {
				hypervisor.RemovePaths(filepath.Join(guestRoot, dir))
			}
		}
	}

	hypervisor.RemovePaths(
		filepath.Join(guestRoot, "var", "lib", "VBoxGuestAdditions"),
		filepath.Join(guestRoot, "opt", "VBoxGuestAdditions"),
	)
	return nil
}
