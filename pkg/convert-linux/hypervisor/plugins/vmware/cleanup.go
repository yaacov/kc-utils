//go:build unix

package vmware

import (
	"bufio"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor"
	"github.com/yaacov/kc-utils/pkg/convert-linux/systemd"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

type Cleanup struct{}

func init() {
	hypervisor.LinuxCleanups.Register("vmware", &Cleanup{})
}

func (c *Cleanup) Detect(guestRoot string) bool {
	indicators := []string{
		filepath.Join(guestRoot, "etc", "vmware-tools"),
		filepath.Join(guestRoot, "usr", "bin", "vmtoolsd"),
		filepath.Join(guestRoot, "usr", "lib", "vmware-tools"),
		filepath.Join(guestRoot, "usr", "bin", "vmware-uninstall-tools.pl"),
	}
	for _, p := range indicators {
		if guestio.FileExists(p) {
			return true
		}
	}
	return false
}

func (c *Cleanup) Cleanup(guestRoot string) error {
	for _, unit := range []string{"vmtoolsd.service", "open-vm-tools.service", "vgauthd.service"} {
		systemd.DisableSystemdUnit(guestRoot, unit)
	}

	disableVMwareRepos(guestRoot)

	systemd.RemovePaths(
		filepath.Join(guestRoot, "etc", "vmware-tools"),
		filepath.Join(guestRoot, "usr", "lib", "vmware-tools"),
		filepath.Join(guestRoot, "usr", "lib64", "vmware-tools"),
	)

	// Schedule package removal via firstboot when RPM DB is present.
	schedulePkgRemove(guestRoot, []string{
		"open-vm-tools", "open-vm-tools-desktop", "VMwareTools",
	})
	return nil
}

func disableVMwareRepos(guestRoot string) {
	reposDir := filepath.Join(guestRoot, "etc", "yum.repos.d")
	entries, err := guestio.FileReadDir(reposDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir || !strings.HasSuffix(e.Name, ".repo") {
			continue
		}
		path := filepath.Join(reposDir, e.Name)
		data, err := guestio.FileRead(path)
		if err != nil {
			continue
		}
		if !strings.Contains(strings.ToLower(string(data)), "vmware.com") {
			continue
		}
		var out strings.Builder
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		inSection := false
		for scanner.Scan() {
			line := scanner.Text()
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "[") {
				inSection = true
			}
			if inSection && (strings.HasPrefix(trimmed, "enabled=") || strings.HasPrefix(trimmed, "enabled =")) {
				out.WriteString("enabled=0\n")
				continue
			}
			out.WriteString(line)
			out.WriteByte('\n')
		}
		if err := guestio.FileWrite(path, []byte(out.String()), 0o644); err != nil {
			slog.Warn("disabling VMware yum repo failed", "path", path, "error", err)
		}
	}
}

func schedulePkgRemove(guestRoot string, pkgs []string) {
	scriptDir := filepath.Join(guestRoot, "var", "lib", "kc-firstboot")
	if err := guestio.FileMkdirAll(scriptDir, 0o755); err != nil {
		slog.Warn("creating VMware pkg-remove script dir failed", "path", scriptDir, "error", err)
	}
	script := filepath.Join(scriptDir, "remove-vmware-pkgs.sh")
	var b strings.Builder
	b.WriteString("#!/bin/bash\n")
	b.WriteString("set -e\n")
	for _, p := range pkgs {
		b.WriteString("rpm -e --nodeps " + p + " 2>/dev/null || dpkg -r " + p + " 2>/dev/null || true\n")
	}
	if err := guestio.FileWrite(script, []byte(b.String()), 0o755); err != nil {
		slog.Warn("writing VMware pkg-remove script failed", "path", script, "error", err)
	}

	unitPath := filepath.Join(guestRoot, "etc", "systemd", "system", "kc-remove-vmware.service")
	unit := `[Unit]
Description=Remove residual VMware packages once
After=local-fs.target

[Service]
Type=oneshot
ExecStart=/var/lib/kc-firstboot/remove-vmware-pkgs.sh
ExecStartPost=/bin/systemctl disable kc-remove-vmware.service
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
`
	if err := guestio.FileMkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
		slog.Warn("creating VMware pkg-remove unit dir failed", "path", filepath.Dir(unitPath), "error", err)
	}
	if err := guestio.FileWrite(unitPath, []byte(unit), 0o644); err != nil {
		slog.Warn("writing VMware pkg-remove unit failed", "path", unitPath, "error", err)
	}
	if err := systemd.EnableSystemdUnit(guestRoot, "kc-remove-vmware.service"); err != nil {
		slog.Warn("enabling VMware pkg-remove unit failed", "error", err)
	}
}
