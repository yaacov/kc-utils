package env

import (
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/yaacov/kc-utils/pkg/backend"
	"github.com/yaacov/kc-utils/pkg/v2v/config"
)

// Load reads V2V_* environment variables and CLI flags.
func Load() (*Config, error) {
	cfg := &Config{
		ExtraArgs:            getExtraArgs(),
		Workdir:              DefaultWorkdir,
		InspectionOutputFile: DefaultInspectionOutputFile,
		LuksDir:              DefaultLuksDir,
		DynamicScriptsDir:    DefaultDynamicScriptsDir,
		MountRoot:            DefaultMountRoot,
		LogLevel:             "info",
		BinDir:               defaultBinDir(),
		CopyConcurrency:      getEnvInt(EnvCopyConcurrency, DefaultCopyConcurrency),
		Backend:              envOr(EnvBackend, backend.NameDirect),
	}

	flag.BoolVar(&cfg.IsLocalMigration, "local-migration", getEnvBool(EnvLocalMigration, true), "local migration mode")
	flag.BoolVar(&cfg.IsInPlace, "in-place", getEnvBool(EnvInPlace, false), "in-place conversion")
	flag.BoolVar(&cfg.OverlayEnabled, "overlay-enabled", getEnvBool(EnvOverlayEnabled, true), "qcow2 overlay")
	flag.BoolVar(&cfg.NbdeClevis, "nbde-clevis", getEnvBool(EnvNbdeClevis, false), "clevis LUKS unlock")
	flag.StringVar(&cfg.Source, "source", os.Getenv(EnvSource), "migration source")
	flag.StringVar(&cfg.DiskPath, "disk-path", os.Getenv(EnvDiskPath), "comma-separated source vmdk paths")
	flag.StringVar(&cfg.Firmware, "firmware", os.Getenv(EnvFirmware), "guest firmware hint (uefi|bios)")
	flag.StringVar(&cfg.LibvirtURL, "libvirt-url", os.Getenv(EnvLibvirtURL), "libvirt URL")
	flag.StringVar(&cfg.VmName, "vm-name", os.Getenv(EnvVmName), "VM name")
	flag.StringVar(&cfg.NewVmName, "new-vm-name", os.Getenv(EnvNewName), "new VM name")
	flag.StringVar(&cfg.RootDisk, "root-disk", envOr(EnvRootDisk, "first"), "root disk selector")
	flag.StringVar(&cfg.StaticIPs, "static-ips", os.Getenv(EnvStaticIPs), "static IP mapping")
	flag.StringVar(&cfg.HostName, "hostname", os.Getenv(EnvHostName), "guest hostname")
	flag.StringVar(&cfg.Workdir, "work-dir", DefaultWorkdir, "working directory")
	flag.StringVar(&cfg.InspectionOutputFile, "inspection-output-file", DefaultInspectionOutputFile, "inspection XML path")
	flag.StringVar(&cfg.LuksDir, "luks-dir", DefaultLuksDir, "LUKS key directory")
	flag.StringVar(&cfg.DynamicScriptsDir, "dynamic-scripts-dir", DefaultDynamicScriptsDir, "dynamic scripts directory")
	flag.BoolVar(&cfg.VsphereVmwareDriverRemoval, "vsphere-vmware-driver-removal", getEnvBool(EnvVsphereVmwareDriverRemoval, false), "VMware driver removal")
	flag.BoolVar(&cfg.WindowsRegistryNetworkConfig, "windows-registry-network-config", getEnvBool(EnvWindowsRegistryNetworkConfig, false), "registry network config")
	flag.BoolVar(&cfg.WaitForGuestReboot, "wait-for-guest-reboot", getEnvBool(EnvWaitForGuestReboot, false), "signal conversion done on COM1")
	flag.BoolVar(&cfg.MultipleIPsPerNic, "multiple-ips-per-nic", getEnvBool(EnvMultipleIPsPerNic, false), "multiple IPs per NIC")
	flag.StringVar(&cfg.Fingerprint, "fingerprint", os.Getenv(EnvFingerprint), "vCenter SSL thumbprint")
	flag.IntVar(&cfg.CopyConcurrency, "copy-concurrency", cfg.CopyConcurrency, "max parallel disk copies")
	flag.BoolVar(&cfg.Offline, "offline", getEnvBool(EnvOffline, false), "pass --offline to converters (use local packages only)")
	flag.StringVar(&cfg.Backend, "backend", envOr(EnvBackend, backend.NameDirect), "guest disk backend (direct|guestfs)")
	flag.Parse()

	cfg.Backend = strings.TrimSpace(cfg.Backend)
	if err := backend.ValidateName(cfg.Backend); err != nil {
		return nil, err
	}

	if err := ValidateCopyMode(cfg); err != nil {
		return nil, err
	}
	if len(cfg.ExtraArgs) > 0 {
		slog.Warn("V2V_extra_args ignored by kc-v2v", "args", cfg.ExtraArgs)
	}
	return cfg, nil
}

func getExtraArgs() []string {
	raw := os.Getenv(EnvExtraArgs)
	if raw == "" {
		return nil
	}
	var args []string
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return nil
	}
	return args
}

func getEnvBool(name string, def bool) bool {
	if v, ok := os.LookupEnv(name); ok {
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			return parsed
		}
	}
	return def
}

func getEnvInt(name string, def int) int {
	if v, ok := os.LookupEnv(name); ok {
		parsed, err := strconv.Atoi(v)
		if err == nil {
			return parsed
		}
	}
	return def
}

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func defaultBinDir() string {
	if dir := os.Getenv("KC_BIN_DIR"); dir != "" {
		return dir
	}
	return "/usr/lib/kc-utils"
}

// LinkCertificates mirrors Forklift virt-v2v entrypoint: when the provider CA
// secret is mounted at /etc/secret/cacert, symlink /opt/ca-bundle.crt → secret.
func LinkCertificates(cfg *Config) error {
	if !IsVSphereSource(cfg) {
		return nil
	}
	if _, err := os.Stat(config.DefaultCaCert); err != nil {
		if os.IsNotExist(err) {
			slog.Debug("no provider CA secret; using system trust store")
			return nil
		}
		return err
	}

	replaced := false
	if err := os.Remove(config.DefaultCaBundle); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		replaced = true
	}
	if err := os.Symlink(config.DefaultCaCert, config.DefaultCaBundle); err != nil {
		return err
	}
	slog.Info("linked vSphere CA bundle",
		"src", config.DefaultCaCert,
		"dest", config.DefaultCaBundle,
		"replaced", replaced,
	)
	return nil
}

// EnsureWorkdir creates the v2v working directory.
func EnsureWorkdir(cfg *Config) error {
	return os.MkdirAll(cfg.Workdir, 0o755)
}

// ProviderCACertMounted reports whether Forklift mounted a provider CA at /etc/secret/cacert.
func ProviderCACertMounted() bool {
	_, err := os.Stat(config.DefaultCaCert)
	return err == nil
}
