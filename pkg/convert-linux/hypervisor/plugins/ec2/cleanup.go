//go:build unix

package ec2

import (
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor"
	"github.com/yaacov/kc-utils/pkg/convert-linux/systemd"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type Cleanup struct{}

func init() {
	hypervisor.LinuxCleanups.Register("ec2", &Cleanup{})
}

const disableDatasourceList = "datasource_list: [None]\n"

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

	return len(cloudInitConfigsWithEc2(guestRoot)) > 0
}

func (c *Cleanup) Cleanup(guestRoot string) error {
	for _, unit := range []string{
		"amazon-ssm-agent.service",
		"amazon-cloudwatch-agent.service",
		"ec2-instance-connect.service",
		"hibagent.service",
		"hibinit-agent.service",
	} {
		systemd.DisableSystemdUnit(guestRoot, unit)
	}

	if err := patchCloudInitDatasources(guestRoot); err != nil {
		slog.Warn("patching cloud-init datasources", "error", err)
	}

	systemd.DisableEC2NetHooks(guestRoot)

	return nil
}

func patchCloudInitDatasources(guestRoot string) error {
	for _, path := range cloudInitConfigsWithEc2(guestRoot) {
		data, err := guest.FileRead(path)
		if err != nil {
			return err
		}
		patched := patchDatasourceListInContent(string(data))
		if err := guest.FileWrite(path, []byte(patched), 0o644); err != nil {
			return err
		}
	}

	cloudCfgDir := filepath.Join(guestRoot, "etc", "cloud", "cloud.cfg.d")
	if err := guest.FileMkdirAll(cloudCfgDir, 0o755); err != nil {
		return err
	}

	disableCfg := filepath.Join(cloudCfgDir, "99-kc-disable-ec2.cfg")
	return guest.FileWrite(disableCfg, []byte(disableDatasourceList), 0o644)
}

// cloudInitConfigsWithEc2 returns the host paths of cloud-init config files
// (cloud.cfg and cloud.cfg.d/*.cfg drop-ins) whose contents reference the Ec2
// datasource. Shared by Detect and patchCloudInitDatasources so a match in
// either cloud.cfg or a drop-in is discovered consistently.
func cloudInitConfigsWithEc2(guestRoot string) []string {
	cloudDir := filepath.Join(guestRoot, "etc", "cloud")
	var matches []string

	cloudCfg := filepath.Join(cloudDir, "cloud.cfg")
	if guest.FileExists(cloudCfg) {
		if data, err := guest.FileRead(cloudCfg); err == nil && strings.Contains(string(data), "Ec2") {
			matches = append(matches, cloudCfg)
		}
	}

	entries, err := guest.FileReadDir(filepath.Join(cloudDir, "cloud.cfg.d"))
	if err != nil {
		return matches
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name, ".cfg") || e.Name == "99-kc-disable-ec2.cfg" {
			continue
		}
		path := filepath.Join(cloudDir, "cloud.cfg.d", e.Name)
		data, err := guest.FileRead(path)
		if err != nil {
			continue
		}
		if strings.Contains(string(data), "Ec2") {
			matches = append(matches, path)
		}
	}
	return matches
}

// patchDatasourceListInContent replaces datasource_list lines that reference Ec2
// with a single-entry None list (cloud-init requires one line, no embedded newlines).
func patchDatasourceListInContent(content string) string {
	var out []string
	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(line, "datasource_list:") && strings.Contains(line, "Ec2") {
			out = append(out, strings.TrimRight(disableDatasourceList, "\n"))
		} else {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
