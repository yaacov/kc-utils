//go:build linux

package ec2

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type Cleanup struct{}

func init() {
	hypervisor.WindowsRemoves.Register("ec2", &Cleanup{})
}

func (c *Cleanup) Detect(mountRoot string, _, _ registry.Hive) bool {
	amazonDir := filepath.Join(mountRoot, "Program Files", "Amazon")
	return guest.FileExists(amazonDir)
}

func (c *Cleanup) Remove(mountRoot string, systemHive, _ registry.Hive) error {
	ccs := currentControlSet(systemHive)
	ec2Services := []string{
		"AWSPVDrivers", "Xennet", "XenVbd", "XenVif", "AWSNVME",
		"AmazonSSMAgent", "AmazonCloudWatchAgent", "Ec2Config", "EC2Launch",
	}
	for _, svc := range ec2Services {
		svcPath := ccs + `\Services\` + svc
		if systemHive.KeyExists(svcPath) {
			systemHive.SetDWORD(svcPath, "Start", 4)
			slog.Info("disabled EC2 service", "service", svc)
		}
	}

	ec2Tasks := []string{
		filepath.Join(mountRoot, "Windows", "System32", "Tasks", "Amazon Ec2 Launch - Instance Integrity"),
		filepath.Join(mountRoot, "Windows", "System32", "Tasks", "Amazon Ec2 Launch - Sysprep"),
	}
	for _, tp := range ec2Tasks {
		_ = guest.FileRemoveAll(tp)
	}

	driversDir := filepath.Join(mountRoot, "Windows", "System32", "drivers")
	entries, err := guest.FileReadDir(driversDir)
	if err == nil {
		for _, e := range entries {
			name := strings.ToLower(e.Name)
			if strings.HasPrefix(name, "xen") && strings.HasSuffix(name, ".sys") {
				_ = guest.FileRemove(filepath.Join(driversDir, e.Name))
			}
		}
	}

	slog.Info("EC2 cleanup complete")
	return nil
}

func currentControlSet(systemHive registry.Hive) string {
	if n, err := systemHive.GetDWORD(`Select`, "Current"); err == nil && n > 0 {
		return fmt.Sprintf("ControlSet%03d", n)
	}
	return "ControlSet001"
}
