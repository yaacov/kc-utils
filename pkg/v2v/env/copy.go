package env

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strings"

	kccopy "github.com/yaacov/kc-utils/pkg/copy"
	v2vtls "github.com/yaacov/kc-utils/pkg/v2v/tls"
	"github.com/yaacov/kc-utils/pkg/v2v/vsphere"
)

// IsVSphereSource reports whether cfg.Source is a vSphere migration.
func IsVSphereSource(cfg *Config) bool {
	return strings.EqualFold(cfg.Source, "vSphere") || strings.EqualFold(cfg.Source, "vmware")
}

// NeedsCopy reports whether kc-v2v should run disk copy before conversion.
func NeedsCopy(cfg *Config) bool {
	return !cfg.IsInPlace
}

// ValidateCopyMode checks that PVC state matches the V2V_inPlace flag.
func ValidateCopyMode(cfg *Config) error {
	targets, err := kccopy.DiscoverTargets()
	if err != nil {
		return fmt.Errorf("discover PVC targets: %w", err)
	}
	if len(targets) == 0 {
		return fmt.Errorf("no PVC targets found at /dev/block* or /mnt/disks/disk*")
	}

	emptyCount := 0
	for _, t := range targets {
		empty, err := kccopy.IsTargetEmpty(t)
		if err != nil {
			return fmt.Errorf("check PVC target %s: %w", t.Path, err)
		}
		if empty {
			emptyCount++
		}
	}
	hasEmpty := emptyCount > 0

	needsCopy := NeedsCopy(cfg)
	slog.Info("copy mode validation",
		"inPlace", cfg.IsInPlace,
		"needsCopy", needsCopy,
		"source", cfg.Source,
		"targets", len(targets),
		"emptyTargets", emptyCount,
		"hasEmpty", hasEmpty,
	)

	if needsCopy {
		if !IsVSphereSource(cfg) {
			return fmt.Errorf("disk copy requires vSphere source (%s=vSphere), got %q", EnvSource, cfg.Source)
		}
		if cfg.LibvirtURL == "" || cfg.VmName == "" {
			return fmt.Errorf("disk copy requires %s and %s", EnvLibvirtURL, EnvVmName)
		}
		if !hasEmpty {
			return fmt.Errorf("%s=0 (copy) but PVC targets are already populated; set %s=1 for pre-filled disks", EnvInPlace, EnvInPlace)
		}
		return nil
	}

	if hasEmpty {
		return fmt.Errorf("%s=1 (no copy) but PVC targets are empty; set %s=0 to run disk copy", EnvInPlace, EnvInPlace)
	}
	return nil
}

// ResolveCopySources returns ordered vmdk paths for disk copy.
func ResolveCopySources(cfg *Config) ([]string, error) {
	if paths := kccopy.SplitDiskPath(cfg.DiskPath); len(paths) > 0 {
		return paths, nil
	}
	if IsVSphereSource(cfg) && cfg.LibvirtURL != "" && cfg.VmName != "" {
		inv, err := vsphere.LoadInventory(cfg)
		if err != nil {
			return nil, fmt.Errorf("vSphere inventory: %w", err)
		}
		return inv.Disks, nil
	}
	return nil, fmt.Errorf("source disk paths required: set V2V_diskPath or vSphere credentials (V2V_libvirtURL, V2V_vmName)")
}

// ValidateCopySourceCount checks that resolved source disks match empty PVC targets.
func ValidateCopySourceCount(sources []string) error {
	targets, err := kccopy.EmptyTargets()
	if err != nil {
		return fmt.Errorf("discover empty PVC targets: %w", err)
	}
	if len(sources) != len(targets) {
		return fmt.Errorf("disk count mismatch: %d source vmdk(s) vs %d empty target(s)", len(sources), len(targets))
	}
	return nil
}

// BuildCopyInput maps Forklift TLS state and copy settings to kc-copy input.
// kc-copy only sees the resulting insecure/ca_cert fields in copy-input.json.
// SourceDisks selects which NFC lease disks kc-copy will stream.
func BuildCopyInput(cfg *Config, sources []string) *kccopy.CopyInput {
	host, datacenter, insecure := parseLibvirtURL(cfg.LibvirtURL)
	in := &kccopy.CopyInput{
		Host:            host,
		Datacenter:      datacenter,
		Insecure:        insecure,
		VMName:          cfg.VmName,
		Fingerprint:     cfg.Fingerprint,
		SourceDisks:     sources,
		Workdir:         cfg.Workdir,
		CopyConcurrency: cfg.CopyConcurrency,
	}
	if ProviderCACertMounted() {
		in.CaCert = DefaultCaCert
	}
	return in
}

func parseLibvirtURL(libvirtURL string) (host, datacenter string, insecure bool) {
	u, err := url.Parse(libvirtURL)
	if err != nil {
		return "", "", false
	}
	host = u.Hostname()
	if p := u.Port(); p != "" {
		host = net.JoinHostPort(host, p)
	}
	insecure = v2vtls.InsecureFromQuery(u.RawQuery)
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) > 0 && parts[0] != "" {
		datacenter = parts[0]
	}
	return host, datacenter, insecure
}
