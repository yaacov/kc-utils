//go:build linux

package convertwindows

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/common/uefi"
	"github.com/yaacov/kc-utils/pkg/convert-windows/crashcontrol"
	"github.com/yaacov/kc-utils/pkg/convert-windows/drivers"
	"github.com/yaacov/kc-utils/pkg/convert-windows/driversource"
	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"
	"github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor"
	"github.com/yaacov/kc-utils/pkg/convert-windows/inspect"
	"github.com/yaacov/kc-utils/pkg/convert-windows/ntfsfix"
	convertoutput "github.com/yaacov/kc-utils/pkg/convert-windows/output"
	"github.com/yaacov/kc-utils/pkg/convert-windows/version"
	"github.com/yaacov/kc-utils/pkg/guest"
)

// Config holds Windows converter pipeline configuration.
type Config struct {
	PrepareData types.PrepareOutput
	Pipeline    *types.PipelineData
	MountRoot   string
	OutputPath  string
	Offline     bool
	StaticIPs   []types.StaticIP
	Backend     string
}

// Run executes the Windows conversion pipeline.
func Run(cfg *Config) error {
	slog.Debug("starting pipeline")

	if cfg.Backend == guest.BackendGuestfs {
		return fmt.Errorf("backend %q is not supported for Windows conversion (requires host mount paths)", cfg.Backend)
	}

	g, err := guest.AttachFromPrepare(cfg.PrepareData.Disks, cfg.PrepareData.RootDevice, cfg.MountRoot, cfg.Backend)
	if err != nil {
		return err
	}
	defer guest.ClearActive()

	output := &types.ConverterOutput{}
	caps := &output.GuestCaps
	caps.Arch = cfg.PrepareData.Inspect.Arch
	if caps.Arch == "" {
		slog.Warn("guest architecture not detected, defaulting to x86_64")
		caps.Arch = "x86_64"
	}

	majorVersion := cfg.PrepareData.Inspect.MajorVersion
	minorVersion := cfg.PrepareData.Inspect.MinorVersion
	osVersion := cfg.PrepareData.Inspect.ProductName
	if osVersion == "" {
		osVersion = fmt.Sprintf("%d.%d", majorVersion, minorVersion)
	}

	ccs := "ControlSet001"
	if cfg.PrepareData.InspectWindows != nil && cfg.PrepareData.InspectWindows.CurrentControlSet > 0 {
		ccs = fmt.Sprintf("ControlSet%03d", cfg.PrepareData.InspectWindows.CurrentControlSet)
	}

	systemGuest, systemHost, systemHive, softwareGuest, softwareHost, softwareHive, err := openHives(g, cfg.PrepareData.InspectWindows)
	if err != nil {
		return err
	}
	defer g.DiscardCheckout(systemHost)
	defer systemHive.Close()
	defer g.DiscardCheckout(softwareHost)
	defer softwareHive.Close()

	// Block 1: Version classification
	versionHandler := version.Classify(&cfg.PrepareData.Inspect)
	slog.Info("matched version handler", "name", versionHandler.Name())

	// Block 2: Locate virtio-win drivers from the pre-extracted directory tree.
	slog.Debug("locating driver source")
	driverFiles, err := driversource.CollectDrivers(caps.Arch, osVersion, versionHandler.Name(),
		versionHandler.DriverOSPreferences(), versionHandler.DriverOSFallbacks())
	if err != nil {
		return fmt.Errorf("locate virtio-win drivers: %w", err)
	}
	if cfg.Offline {
		slog.Info("offline mode enabled, skipping network-dependent operations")
	}

	// Block 2b: Detect antivirus
	slog.Debug("detecting antivirus")
	output.Warnings = append(output.Warnings, inspect.DetectAntivirus(softwareHive)...)

	// Block 3: Detect RTC mode
	slog.Debug("detecting RTC mode")
	inspect.DetectRTCMode(systemHive, ccs, caps)

	// Block 4 (pluggable): Remove hypervisor software
	slog.Info("removing hypervisor software", "plugins", hypervisor.WindowsRemoves.List())
	for name, u := range hypervisor.WindowsRemoves.All() {
		if u.Detect(cfg.MountRoot, systemHive, softwareHive) {
			slog.Info("running hypervisor remove", "name", name)
			if uninstErr := u.Remove(cfg.MountRoot, systemHive, softwareHive); uninstErr != nil {
				slog.Warn("hypervisor remove failed", "name", name, "error", uninstErr)
				output.Errors = append(output.Errors, types.BlockError{
					Block: "hypervisor-remove/" + name, Message: uninstErr.Error(),
				})
				continue
			}
			slog.Info("hypervisor remove complete", "name", name)
		}
	}

	// Block 5: Copy virtio drivers into the guest
	slog.Debug("copying virtio drivers")
	copiedDriverNames, err := drivers.Copy(cfg.MountRoot, driverFiles)
	if err != nil {
		output.Errors = append(output.Errors, types.BlockError{
			Block: "driver-copy", Message: err.Error(),
		})
	}

	// Block 6 (pluggable): Register drivers in registry
	slog.Debug("registering drivers")
	registerDrivers(systemHive, ccs, copiedDriverNames, versionHandler, caps.Arch)

	// Block 7: Update DevicePath registry key
	slog.Debug("updating device path")
	drivers.Update(softwareHive)

	// Block 8 (pluggable): Disable hypervisor services
	slog.Info("disabling hypervisor services", "plugins", hypervisor.WindowsServiceDisablers.List())
	for name, u := range hypervisor.WindowsServiceDisablers.All() {
		if u.Detect(cfg.MountRoot, systemHive, ccs) {
			slog.Info("running hypervisor service disable", "name", name)
			if uncErr := u.DisableServices(cfg.MountRoot, systemHive, ccs); uncErr != nil {
				slog.Warn("hypervisor service disable failed", "name", name, "error", uncErr)
				output.Errors = append(output.Errors, types.BlockError{
					Block: "hypervisor-service/" + name, Message: uncErr.Error(),
				})
				continue
			}
			slog.Info("hypervisor service disable complete", "name", name)
		}
	}

	// Block 9: Disable crash auto-reboot on BSOD
	slog.Debug("disabling crash auto-reboot")
	crashcontrol.Disable(systemHive, ccs)

	// Blocks 10–12: Firstboot scripts
	slog.Debug("generating firstboot scripts")
	if err := firstboot.Configure(&firstboot.Config{
		MountRoot:   cfg.MountRoot,
		Offline:     cfg.Offline,
		DriverFiles: driverFiles,
		StaticIPs:   cfg.StaticIPs,
		Options:     cfg.PrepareData.Options,
		Version:     versionHandler,
	}, softwareHive); err != nil {
		output.Errors = append(output.Errors, types.BlockError{
			Block: "firstboot", Message: err.Error(),
		})
	}

	// Block 13: NTFS boot sector fix
	slog.Debug("NTFS heads fix")
	ntfsfix.Fix(versionHandler.NeedsNTFSHeadsFix(), cfg.PrepareData.Disks)

	// Block 14 (pluggable): UEFI BCD fixup on ESP partitions
	slog.Debug("UEFI BCD fixup")
	if cfg.PrepareData.Firmware.Type == "uefi" {
		output.Errors = append(output.Errors, uefi.ConvertAllESPs(cfg.MountRoot, cfg.PrepareData.Firmware.ESPDevices)...)
	}

	// Block 15: Build output guest capabilities
	slog.Debug("building guest capabilities")
	convertoutput.Build(caps, copiedDriverNames)

	// Block 17: Post-convert permission fixup
	slog.Debug("post-convert fixup")
	convertoutput.FixPermissions(cfg.MountRoot)

	if err := systemHive.Save(); err != nil {
		return fmt.Errorf("saving SYSTEM hive: %w", err)
	}
	if err := g.Checkin(systemGuest, systemHost); err != nil {
		return fmt.Errorf("checkin SYSTEM hive: %w", err)
	}
	if err := softwareHive.Save(); err != nil {
		return fmt.Errorf("saving SOFTWARE hive: %w", err)
	}
	if err := g.Checkin(softwareGuest, softwareHost); err != nil {
		return fmt.Errorf("checkin SOFTWARE hive: %w", err)
	}

	slog.Debug("writing output")
	cfg.Pipeline.Convert = output
	if err := types.WriteJSON(cfg.OutputPath, cfg.Pipeline); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	slog.Info("pipeline complete")
	return nil
}

func openHives(g *guest.Guest, iw *types.WindowsInspect) (systemGuest, systemHost string, systemHive registry.Hive, softwareGuest, softwareHost string, softwareHive registry.Hive, err error) {
	editor, ok := registry.Editors.Get("hivex")
	if !ok {
		err = fmt.Errorf("registry editor 'hivex' not registered")
		return
	}

	systemGuest = "/Windows/System32/config/SYSTEM"
	if iw != nil && iw.SystemHive != "" {
		systemGuest = "/" + filepath.ToSlash(strings.TrimPrefix(iw.SystemHive, "/"))
	}
	systemHost, err = g.Checkout(systemGuest)
	if err != nil {
		err = fmt.Errorf("checkout SYSTEM hive: %w", err)
		return
	}
	systemHive, err = editor.OpenHive(systemHost)
	if err != nil {
		g.DiscardCheckout(systemHost)
		err = fmt.Errorf("opening SYSTEM hive: %w", err)
		return
	}

	softwareGuest = "/Windows/System32/config/SOFTWARE"
	if iw != nil && iw.SoftwareHive != "" {
		softwareGuest = "/" + filepath.ToSlash(strings.TrimPrefix(iw.SoftwareHive, "/"))
	}
	softwareHost, err = g.Checkout(softwareGuest)
	if err != nil {
		systemHive.Close()
		g.DiscardCheckout(systemHost)
		err = fmt.Errorf("checkout SOFTWARE hive: %w", err)
		return
	}
	softwareHive, err = editor.OpenHive(softwareHost)
	if err != nil {
		g.DiscardCheckout(softwareHost)
		systemHive.Close()
		g.DiscardCheckout(systemHost)
		err = fmt.Errorf("opening SOFTWARE hive: %w", err)
		return
	}
	return
}

func registerDrivers(systemHive registry.Hive, ccs string, copiedDriverNames []string, vh version.VersionHandler, arch string) {
	storageDrivers := []string{"viostor", "vioscsi"}
	for _, drvName := range storageDrivers {
		found := false
		for _, copied := range copiedDriverNames {
			if strings.EqualFold(copied, drvName) {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		driverPath := fmt.Sprintf(`system32\drivers\%s.sys`, drvName)
		regName := vh.DriverRegistrarName()
		registrar, regOK := drivers.Registrars.Get(regName)
		if !regOK {
			slog.Warn("driver registrar not found", "registrar", regName)
			continue
		}
		if regErr := registrar.Register(systemHive, ccs, drvName, driverPath, "SCSI miniport", arch); regErr != nil {
			slog.Warn("registering driver failed", "driver", drvName, "error", regErr)
		} else {
			slog.Info("registered driver", "driver", drvName, "registrar", regName)
		}
	}
}
