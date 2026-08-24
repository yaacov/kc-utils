package env

import (
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

// BuildPrepareInput constructs PrepareInput from v2v config and discovered disks.
func BuildPrepareInput(cfg *Config, disks []DiskInfo, source types.SourceSpec) (types.PrepareInput, error) {
	staticIPs, err := ParseStaticIPs(cfg.StaticIPs)
	if err != nil {
		return types.PrepareInput{}, err
	}
	luksSpec, err := BuildLUKSSpec(cfg)
	if err != nil {
		return types.PrepareInput{}, err
	}
	bitlkSpec, err := BuildBitLockerSpec(cfg)
	if err != nil {
		return types.PrepareInput{}, err
	}

	var diskSpecs []types.DiskSpec
	for _, d := range disks {
		diskSpecs = append(diskSpecs, types.DiskSpec{Path: d.Path, Format: "raw"})
	}

	root := cfg.RootDisk
	if root == "" {
		root = "first"
	}

	return types.PrepareInput{
		Disks:     diskSpecs,
		Source:    source,
		LUKS:      luksSpec,
		BitLocker: bitlkSpec,
		Options: types.PrepareOptions{
			TmpDir:                 cfg.Workdir,
			StaticIPs:              staticIPs,
			Root:                   root,
			Hostname:               cfg.HostName,
			DynamicScriptsDir:      cfg.DynamicScriptsDir,
			VMwareDriverRemoval:    cfg.VsphereVmwareDriverRemoval,
			WindowsRegistryNetwork: cfg.WindowsRegistryNetworkConfig,
			MultipleIPsPerNic:      cfg.MultipleIPsPerNic,
			WaitForGuestReboot:     cfg.WaitForGuestReboot,
		},
	}, nil
}

func SourceName(cfg *Config) string {
	if cfg.VmName != "" {
		return cfg.VmName
	}
	return "disk"
}

func NormalizeSourceType(source string) string {
	switch strings.ToLower(source) {
	case "ec2", "hyperv", "ova", "vsphere":
		return strings.ToLower(source)
	case "nutanix", "nutanix-ahv":
		return "nutanix"
	default:
		if source == "" {
			return "disk"
		}
		return source
	}
}
