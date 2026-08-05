//go:build linux

package ec2

import (
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type Cleanup struct{}

func init() {
	hypervisor.LinuxCleanups.Register("ec2", &Cleanup{})
}

func (c *Cleanup) Detect(guestRoot string) bool {
	indicators := []string{
		filepath.Join(guestRoot, "usr", "bin", "amazon-ssm-agent"),
		filepath.Join(guestRoot, "etc", "amazon", "ssm"),
		filepath.Join(guestRoot, "usr", "bin", "ec2-instance-connect"),
	}
	for _, p := range indicators {
		if guest.FileExists(p) {
			return true
		}
	}

	cloudCfg := filepath.Join(guestRoot, "etc", "cloud", "cloud.cfg")
	if guest.FileExists(cloudCfg) {
		data, err := guest.FileRead(cloudCfg)
		if err == nil && strings.Contains(string(data), "Ec2") {
			return true
		}
	}

	return false
}

func (c *Cleanup) Cleanup(guestRoot string) error {
	services := []string{
		"amazon-ssm-agent.service",
		"amazon-cloudwatch-agent.service",
		"ec2-instance-connect.service",
	}
	symlinkDirs := []string{
		filepath.Join(guestRoot, "etc", "systemd", "system", "multi-user.target.wants"),
		filepath.Join(guestRoot, "etc", "systemd", "system", "default.target.wants"),
	}
	for _, dir := range symlinkDirs {
		for _, svc := range services {
			_ = guest.FileRemove(filepath.Join(dir, svc))
		}
	}

	cloudCfgDir := filepath.Join(guestRoot, "etc", "cloud", "cloud.cfg.d")
	if guest.FileExists(cloudCfgDir) {
		disableCfg := filepath.Join(cloudCfgDir, "99-kc-disable-ec2.cfg")
		content := "datasource_list: [None]\n"
		if err := guest.FileWrite(disableCfg, []byte(content), 0o644); err != nil {
			slog.Warn("writing cloud-init disable config", "error", err)
		}
	}

	return nil
}
