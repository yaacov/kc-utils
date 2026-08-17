//go:build unix

package qemuga

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/convert-linux/guestagent"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

type QEMUAgent struct{}

func init() {
	guestagent.Agents.Register("qemu-ga", &QEMUAgent{})
}

func (q *QEMUAgent) Detect(guestRoot string) bool {
	p := filepath.Join(guestRoot, "usr", "bin", "qemu-ga")
	return guestio.FileExists(p)
}

func (q *QEMUAgent) Remove(guestRoot string) error {
	// Check if RPM-based: log that package removal would happen at firstboot.
	rpmDB := filepath.Join(guestRoot, "var", "lib", "rpm")
	if guestio.FileExists(rpmDB) {
		slog.Info("RPM database found, qemu-guest-agent will be removed via rpm -e at firstboot")
	}

	// Check if dpkg-based: log that package removal would happen at firstboot.
	dpkgDB := filepath.Join(guestRoot, "var", "lib", "dpkg")
	if guestio.FileExists(dpkgDB) {
		slog.Info("dpkg database found, qemu-guest-agent will be removed via dpkg --purge at firstboot")
	}

	// Remove the binary.
	binaryPath := filepath.Join(guestRoot, "usr", "bin", "qemu-ga")
	if err := guestio.FileRemove(binaryPath); err != nil && !os.IsNotExist(err) {
		slog.Warn("failed to remove binary", "path", binaryPath, "error", err)
	} else if err == nil {
		slog.Info("removed binary", "path", binaryPath)
	}

	// Remove the systemd service file if it exists.
	servicePath := filepath.Join(guestRoot, "usr", "lib", "systemd", "system", "qemu-guest-agent.service")
	if err := guestio.FileRemove(servicePath); err != nil && !os.IsNotExist(err) {
		slog.Warn("failed to remove service file", "path", servicePath, "error", err)
	} else if err == nil {
		slog.Info("removed service file", "path", servicePath)
	}

	return nil
}
