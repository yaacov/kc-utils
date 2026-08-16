//go:build unix

package guestcleanup

import (
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/configedit/modprobe"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

// staleModules lists hypervisor kernel modules whose modprobe aliases
// should be removed during conversion.
var staleModules = map[string]bool{
	"vmw_pvscsi":    true,
	"vmxnet3":       true,
	"vmxnet":        true,
	"hv_vmbus":      true,
	"hv_storvsc":    true,
	"hv_netvsc":     true,
	"xen_blkfront":  true,
	"xen_netfront":  true,
	"vboxguest":     true,
	"vboxsf":        true,
	"vboxvideo":     true,
	"prl_tg":        true,
	"prl_eth":       true,
	"prl_fs":        true,
	"prl_fs_freeze": true,
}

// Configure writes virtio module aliases to modprobe.d and removes
// stale hypervisor aliases from existing config files.
func Configure(guestRoot string) {
	modprobeDir := filepath.Join(guestRoot, "etc", "modprobe.d")
	cleanStaleAliases(modprobeDir)

	modprobePath := filepath.Join(modprobeDir, "kc-virtio.conf")
	modCfg := modprobe.Parse("")
	if existingModprobe, err := guestio.FileRead(modprobePath); err == nil {
		modCfg = modprobe.Parse(string(existingModprobe))
	}
	modCfg.AddAlias("scsi_hostadapter", "virtio_blk")
	modCfg.AddAlias("scsi_hostadapter1", "virtio_scsi")
	modCfg.AddAlias("eth0", "virtio_net")
	if err := guestio.FileMkdirAll(modprobeDir, 0o755); err != nil {
		slog.Warn("creating modprobe.d failed", "error", err)
	} else if err := guestio.FileWrite(modprobePath, []byte(modCfg.String()), 0o644); err != nil {
		slog.Warn("writing modprobe config failed", "error", err)
	}
}

// cleanStaleAliases removes alias lines that reference hypervisor modules
// from all .conf files in modprobe.d.
func cleanStaleAliases(modprobeDir string) {
	entries, err := guestio.FileReadDir(modprobeDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir || !strings.HasSuffix(entry.Name, ".conf") {
			continue
		}
		if entry.Name == "kc-virtio.conf" {
			continue
		}
		filePath := filepath.Join(modprobeDir, entry.Name)
		data, err := guestio.FileRead(filePath)
		if err != nil {
			continue
		}
		cleaned, changed := removeStaleLines(string(data))
		if changed {
			if err := guestio.FileWrite(filePath, []byte(cleaned), 0o644); err != nil {
				slog.Warn("cleaning modprobe config failed", "file", entry.Name, "error", err)
			} else {
				slog.Info("cleaned stale aliases from modprobe config", "file", entry.Name)
			}
		}
	}
}

func removeStaleLines(content string) (string, bool) {
	lines := strings.Split(content, "\n")
	var result []string
	changed := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isStaleDirective(trimmed) {
			changed = true
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n"), changed
}

func isStaleDirective(line string) bool {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return false
	}
	directive := fields[0]
	switch directive {
	case "alias":
		if len(fields) >= 3 {
			return staleModules[fields[2]]
		}
	case "install", "options", "blacklist":
		return staleModules[fields[1]]
	}
	return false
}
