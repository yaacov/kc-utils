//go:build unix

package finalize

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/finalize/customize"
	"github.com/yaacov/kc-utils/pkg/finalize/metadata"
	"github.com/yaacov/kc-utils/pkg/finalize/target"
	"github.com/yaacov/kc-utils/pkg/guest"
)

// Config holds finalizer pipeline configuration.
type Config struct {
	Pipeline   *types.PipelineData
	MountRoot  string
	OutputPath string
	Backend    string
}

// Run executes the finalizer pipeline.
func Run(cfg *Config) error {
	slog.Debug("starting pipeline")

	meta := &types.TargetMeta{
		GuestCaps:      cfg.Pipeline.Convert.GuestCaps,
		BootDevice:     cfg.Pipeline.Prepare.BootDevice,
		Disks:          cfg.Pipeline.Prepare.Disks,
		Inspect:        cfg.Pipeline.Prepare.Inspect,
		TargetFirmware: cfg.Pipeline.Prepare.Firmware.Type,
	}

	g, err := guest.AttachFromPrepare(cfg.Pipeline.Prepare.Disks, cfg.Pipeline.Prepare.RootDevice, cfg.MountRoot, cfg.Backend)
	if err != nil {
		return err
	}
	defer guest.ClearActive()

	// Ensure resources are reclaimed if we return before explicit release
	// (unmount → fsck → release devices).
	tornDown := false
	defer func() {
		if tornDown {
			return
		}
		slog.Info("tearing down guest (deferred)", "backend", g.BackendName(), "mountRoot", cfg.MountRoot)
		if terr := g.Teardown(); terr != nil {
			slog.Warn("guest teardown failed", "error", terr)
		}
	}()

	customizers := customize.Customizers.List()
	slog.Info("running customizations", "plugins", customizers)
	customizerOpts := metadata.CustomizerOpts(cfg.Pipeline)
	for _, name := range customizers {
		c, ok := customize.Customizers.Get(name)
		if !ok {
			continue
		}
		slog.Info("running customizer", "name", name)
		if err := c.Apply(cfg.MountRoot, customizerOpts); err != nil {
			slog.Warn("customization failed", "name", name, "error", err)
			meta.Warnings = append(meta.Warnings,
				fmt.Sprintf("customization %s failed: %v", name, err))
			continue
		}
		slog.Info("customizer complete", "name", name)
	}

	// Sync is a no-op (direct mounts and guestfs appliance writes are already live).
	slog.Info("syncing guest tree to disks", "backend", g.BackendName())
	if err := g.Sync(); err != nil {
		slog.Warn("guest sync failed", "error", err)
		meta.Warnings = append(meta.Warnings, fmt.Sprintf("guest sync failed: %v", err))
	}

	trimPaths := mountedGuestPaths(cfg.MountRoot, cfg.Pipeline.Prepare.Disks)
	slog.Info("trimming filesystems", "backend", g.BackendName(), "mounts", len(trimPaths))
	for _, mountpoint := range trimPaths {
		if err := g.FSTrim(mountpoint); err != nil {
			slog.Warn("fstrim failed", "mountpoint", mountpoint, "error", err)
		}
	}

	slog.Info("unmounting guest filesystems", "backend", g.BackendName(), "mountRoot", cfg.MountRoot)
	if err := g.UnmountFilesystems(); err != nil {
		slog.Warn("guest unmount failed", "error", err)
	}

	slog.Info("checking filesystems", "backend", g.BackendName())
	for _, d := range cfg.Pipeline.Prepare.Disks {
		for _, p := range d.Partitions {
			if err := g.FSCheck(p.DevicePath, p.FSType); err != nil {
				slog.Warn("fscheck failed", "device", p.DevicePath, "error", err)
				meta.Warnings = append(meta.Warnings,
					fmt.Sprintf("filesystem check failed on %s: %v", p.DevicePath, err))
			}
		}
	}

	slog.Info("releasing guest devices", "backend", g.BackendName(), "mountRoot", cfg.MountRoot)
	if err := g.ReleaseDevices(); err != nil {
		slog.Warn("guest release failed", "error", err)
	}
	tornDown = true

	slog.Debug("resolving firmware")
	meta.TargetFirmware = target.Target(meta.TargetFirmware)

	slog.Debug("assigning bus slots")
	meta.TargetBuses = target.Buses(cfg.Pipeline.Prepare.Disks, cfg.Pipeline.Convert.GuestCaps.BlockBus)
	meta.TargetNICs = target.NICs(cfg.Pipeline.Prepare.Source.NICs, cfg.Pipeline.Convert.GuestCaps.NetBus)

	slog.Debug("writing metadata")
	cfg.Pipeline.Target = meta
	if err := metadata.WriteTargetMeta(cfg.OutputPath, cfg.Pipeline); err != nil {
		return err
	}

	slog.Info("pipeline complete")
	return nil
}

// TeardownOnly reclaims orphaned guest resources without Sync, customize,
// trim, or metadata writes. Used by kc-v2v on pipeline failure.
func TeardownOnly(cfg *Config) error {
	slog.Info("teardown-only starting", "backend", cfg.Backend, "mountRoot", cfg.MountRoot)

	if cfg.Pipeline != nil && cfg.Pipeline.Prepare != nil && len(cfg.Pipeline.Prepare.Disks) > 0 {
		disks := guest.WithRootMount(cfg.Pipeline.Prepare.Disks, cfg.Pipeline.Prepare.RootDevice)
		diskSpecs := types.DiskSpecsFrom(disks)
		g, err := guest.AttachMounted(diskSpecs, cfg.MountRoot, cfg.Backend, disks)
		if err != nil {
			slog.Warn("attach for teardown-only failed; falling back to mount-root cleanup", "error", err)
			return guest.TeardownMountRoot(cfg.MountRoot, cfg.Backend)
		}
		guest.SetActive(g)
		defer guest.ClearActive()
		if err := g.TeardownDiscard(); err != nil {
			return fmt.Errorf("teardown discard: %w", err)
		}
		slog.Info("teardown-only complete", "backend", cfg.Backend)
		return nil
	}

	if err := guest.TeardownMountRoot(cfg.MountRoot, cfg.Backend); err != nil {
		return fmt.Errorf("teardown mount root: %w", err)
	}
	slog.Info("teardown-only complete (mount-root only)", "backend", cfg.Backend)
	return nil
}

func mountedGuestPaths(mountRoot string, disks []types.DiskInfo) []string {
	seen := map[string]bool{mountRoot: true}
	paths := []string{mountRoot}
	for _, d := range disks {
		for _, p := range d.Partitions {
			mp := strings.TrimSpace(p.MountPoint)
			if mp == "" || mp == "/" {
				continue
			}
			hostPath := filepath.Join(mountRoot, strings.TrimPrefix(mp, "/"))
			if seen[hostPath] {
				continue
			}
			seen[hostPath] = true
			paths = append(paths, hostPath)
		}
	}
	sort.Strings(paths)
	return paths
}
