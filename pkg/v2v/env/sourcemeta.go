package env

import (
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/v2v/config"
	"github.com/yaacov/kc-utils/pkg/v2v/vsphere"
)

// FetchSourceMeta loads NIC and firmware metadata from env vars or vCenter.
func FetchSourceMeta(cfg *config.Config) (types.SourceSpec, error) {
	source := types.SourceSpec{
		Name: SourceName(cfg),
		Type: NormalizeSourceType(cfg.Source),
	}
	source.FirmwareHint = firmwareHintFromEnv(cfg.Firmware)

	if isVSphereSource(cfg) && cfg.LibvirtURL != "" && cfg.VmName != "" {
		inv, err := vsphere.LoadInventory(cfg)
		if err != nil {
			return source, err
		}
		if len(inv.NICs) > 0 {
			source.NICs = inv.NICs
		}
		if source.FirmwareHint == "" && inv.FirmwareHint != "" {
			source.FirmwareHint = inv.FirmwareHint
		}
		if cfg.HostName == "" && inv.HostName != "" {
			cfg.HostName = inv.HostName
		}
	}

	if source.FirmwareHint == "" {
		source.FirmwareHint = "bios"
	}
	return source, nil
}

func isVSphereSource(cfg *config.Config) bool {
	switch strings.ToLower(cfg.Source) {
	case "vsphere", "vmware":
		return true
	default:
		return false
	}
}

func firmwareHintFromEnv(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "efi", "uefi":
		return "uefi"
	case "bios":
		return "bios"
	default:
		return ""
	}
}
