//go:build unix

package convertlinux

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/firstboot"
	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/common/uefi"
	"github.com/yaacov/kc-utils/pkg/convert-linux/bootconfig"
	"github.com/yaacov/kc-utils/pkg/convert-linux/bootloader"
	"github.com/yaacov/kc-utils/pkg/convert-linux/distro"
	"github.com/yaacov/kc-utils/pkg/convert-linux/guestagent"
	"github.com/yaacov/kc-utils/pkg/convert-linux/guestcaps"
	"github.com/yaacov/kc-utils/pkg/convert-linux/guestcleanup"
	"github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor"
	"github.com/yaacov/kc-utils/pkg/convert-linux/initramfs"
	"github.com/yaacov/kc-utils/pkg/convert-linux/kernel"
	"github.com/yaacov/kc-utils/pkg/convert-linux/network/networkd"
	"github.com/yaacov/kc-utils/pkg/convert-linux/network/staticip"
	"github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming"
	"github.com/yaacov/kc-utils/pkg/convert-linux/remap"
	"github.com/yaacov/kc-utils/pkg/convert-linux/selinux"
	"github.com/yaacov/kc-utils/pkg/guest"
)

// Config holds linux converter pipeline configuration.
type Config struct {
	PrepareData types.PrepareOutput
	Pipeline    *types.PipelineData
	MountRoot   string
	OutputPath  string
	Offline     bool
	Backend     string
}

// Run executes the Linux conversion pipeline.
func Run(cfg *Config) error {
	slog.Debug("starting pipeline")

	_, err := guest.AttachFromPrepare(cfg.PrepareData.Disks, cfg.PrepareData.RootDevice, cfg.MountRoot, cfg.Backend)
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

	// Block 1 (pluggable): Distro classification
	slog.Debug("classifying distro")
	var distroHandler distro.DistroHandler
	for name, handler := range distro.Handlers.All() {
		if handler.Matches(&cfg.PrepareData.Inspect) {
			slog.Info("matched distro handler", "name", name)
			distroHandler = handler
			break
		}
	}
	if distroHandler == nil {
		switch cfg.PrepareData.Inspect.Distro {
		case "alt":
			slog.Warn("ALT Linux detected but distro handler not implemented, using defaults",
				"distro", cfg.PrepareData.Inspect.Distro)
		default:
			slog.Warn("no distro handler matched, using defaults",
				"distro", cfg.PrepareData.Inspect.Distro)
		}
	}

	// Block 2: Package format
	slog.Debug("determining package format")
	pkgFormat := distro.Format(cfg.PrepareData.Inspect.Distro)

	// Block 3: Package manager
	slog.Debug("determining package manager")
	pkgManager := distro.Name(cfg.PrepareData.Inspect.Distro)

	// Block 4 (pluggable): Bootloader detection (BLS before grub2)
	slog.Debug("detecting bootloader")
	blName, activeHandler := bootloader.DetectFirst(cfg.MountRoot)
	if activeHandler != nil {
		slog.Info("found bootloader", "name", blName)
	}

	// Block 5 (pluggable): Kernel inspection
	slog.Debug("scanning kernels")
	allKernels := scanKernels(cfg.MountRoot)

	// Block 6 (pluggable): Device remapping
	slog.Info("remapping devices", "plugins", remap.Remappers.List())
	for name, remapper := range remap.Remappers.All() {
		if remapper.Detect(cfg.MountRoot) {
			slog.Info("running device remapper", "name", name)
			if err := remapper.Remap(cfg.MountRoot); err != nil {
				slog.Warn("device remap failed", "name", name, "error", err)
				output.Errors = append(output.Errors, types.BlockError{
					Block: "device-remap/" + name, Message: err.Error(),
				})
				continue
			}
			slog.Info("device remapper complete", "name", name)
		}
	}
	slog.Debug("NIC remapping is environment-specific; skipping in-guest remap")

	// Block 7 (pluggable): UEFI fixup
	slog.Debug("UEFI fixup")
	output.Errors = append(output.Errors, fixupUEFI(cfg.MountRoot, cfg.PrepareData.Firmware)...)

	// Block 8: Kernel selection
	slog.Debug("finalizing kernel selection")
	selectedKernel := selectKernelAndSetDefault(cfg.MountRoot, allKernels, activeHandler)

	// Block 9: Console configuration
	slog.Debug("configuring console")
	bootconfig.ConfigureConsole(cfg.MountRoot, activeHandler, distroHandler)

	// Block 10: Display configuration
	slog.Debug("configuring display")
	bootconfig.ConfigureDisplay(cfg.MountRoot, activeHandler)
	bootconfig.ConfigureXorgDriver(cfg.MountRoot)

	// Block 11 (pluggable): Hypervisor cleanup
	slog.Info("cleaning up hypervisor artifacts", "plugins", hypervisor.LinuxCleanups.List())
	for name, u := range hypervisor.LinuxCleanups.All() {
		if u.Detect(cfg.MountRoot) {
			slog.Info("running hypervisor cleanup", "name", name)
			if err := u.Cleanup(cfg.MountRoot); err != nil {
				slog.Warn("hypervisor cleanup failed", "name", name, "error", err)
				output.Errors = append(output.Errors, types.BlockError{
					Block: "hypervisor-cleanup/" + name, Message: err.Error(),
				})
				continue
			}
			slog.Info("hypervisor cleanup complete", "name", name)
		}
	}

	networkdPrimary := networkd.Detect(cfg.MountRoot)
	output.Errors = append(output.Errors, applyKubeVirtNetworking(cfg.MountRoot, networkdPrimary)...)

	// Block 12: Guest agent installation
	slog.Debug("installing guest agent")
	guestagent.Install(
		cfg.MountRoot,
		pkgFormat,
		pkgManager,
		caps.Arch,
		cfg.PrepareData.Inspect.Distro,
		cfg.PrepareData.Inspect.MajorVersion,
		cfg.Offline,
	)

	// Block 13: Guest cleanup
	slog.Debug("cleaning guest artifacts")
	guestcleanup.Run(cfg.MountRoot)

	// Block 14: Initramfs injection — rebuild to ensure virtio drivers are
	// included in the initramfs (on-disk modules alone are not sufficient for
	// early boot). Matching virt-v2v which always rebuilds for the best kernel.
	if selectedKernel != nil {
		slog.Info("initramfs rebuild")
		if err := initramfs.InjectVirtioModules(cfg.MountRoot, selectedKernel); err != nil {
			return fmt.Errorf("initramfs rebuild failed (VM will not boot without virtio drivers): %w", err)
		}
	} else {
		slog.Warn("skipping initramfs rebuild, no kernel selected")
	}

	// Block 15: Static IP configuration + NIC naming preservation
	if len(cfg.PrepareData.Options.StaticIPs) > 0 {
		output.Errors = append(output.Errors, configureStaticIPs(cfg.MountRoot, cfg.PrepareData.Options.StaticIPs, networkdPrimary)...)
	}

	// Block 16: Offline SELinux relabel — run setfiles against the guest
	// filesystem now so the guest doesn't need a slow boot-time relabel +
	// automatic reboot via /.autorelabel.
	slog.Debug("SELinux relabel")
	mountPoints := guestMountPoints(cfg.PrepareData.Disks)
	if relabeled, err := selinux.Relabel(cfg.MountRoot, mountPoints); err != nil {
		slog.Warn("offline SELinux relabel failed, finalize will create /.autorelabel as fallback", "error", err)
	} else if relabeled {
		output.SELinuxRelabeled = true
	}

	// Block 17: Build GuestCaps
	slog.Debug("building guest capabilities")
	guestcaps.Build(caps, selectedKernel)

	// Write output
	slog.Debug("writing output")
	cfg.Pipeline.Convert = output
	if err := types.WriteJSON(cfg.OutputPath, cfg.Pipeline); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	slog.Info("pipeline complete")
	return nil
}

func guestMountPoints(disks []types.DiskInfo) []string {
	seen := map[string]bool{"/": true}
	mps := []string{"/"}
	for _, d := range disks {
		for _, p := range d.Partitions {
			mp := strings.TrimSpace(p.MountPoint)
			if mp == "" || mp == "/" {
				continue
			}
			if !seen[mp] {
				seen[mp] = true
				mps = append(mps, mp)
			}
		}
	}
	return mps
}

func fixupUEFI(mountRoot string, firmware types.FirmwareInfo) []types.BlockError {
	if types.FirmwareType(firmware.Type) != types.FirmwareUEFI {
		return nil
	}
	efiPath := filepath.Join(mountRoot, "boot", "efi")
	if _, freeInodes, err := guest.FileStatFS(efiPath); err == nil && freeInodes >= 0 && freeInodes < 10 {
		slog.Warn("EFI partition has insufficient free inodes, skipping UEFI fixup", "path", efiPath, "freeInodes", freeInodes)
		return []types.BlockError{
			{Block: "uefi/pre-check", Message: fmt.Sprintf("EFI partition has %d free inodes", freeInodes)},
		}
	}
	return uefi.ConvertAllESPs(mountRoot, firmware.ESPDevices)
}

func selectKernelAndSetDefault(mountRoot string, allKernels []types.KernelInfo, activeHandler bootloader.BootloaderHandler) *types.KernelInfo {
	var selected *types.KernelInfo
	if len(allKernels) > 0 {
		selected = kernel.Best(allKernels)
	}
	if selected == nil {
		slog.Warn("no bootable kernel with virtio support found, defaulting to virtio caps", "scanned", len(allKernels))
		return nil
	}
	slog.Info("selected kernel", "version", selected.Version)
	if activeHandler != nil {
		if err := activeHandler.SetDefaultKernel(mountRoot, selected.Version); err != nil {
			slog.Warn("setting default kernel failed", "error", err)
		}
	}
	return selected
}

func applyKubeVirtNetworking(mountRoot string, networkdPrimary bool) []types.BlockError {
	if !networkdPrimary {
		return nil
	}
	slog.Info("configuring systemd-networkd for KubeVirt")
	var errs []types.BlockError
	if err := networkd.InstallKubeVirtNetworking(mountRoot); err != nil {
		slog.Warn("systemd-networkd KubeVirt networking failed", "error", err)
		errs = append(errs, types.BlockError{
			Block: "networkd/kubevirt", Message: err.Error(),
		})
	}
	return errs
}

func configureStaticIPs(mountRoot string, staticIPs []types.StaticIP, networkdPrimary bool) []types.BlockError {
	var errs []types.BlockError

	if networkdPrimary {
		slog.Debug("configuring static IPs via systemd-networkd")
		if err := networkd.WriteStaticNetworks(mountRoot, staticIPs); err != nil {
			slog.Warn("writing systemd-networkd static config failed", "error", err)
			errs = append(errs, types.BlockError{
				Block: "static-ip/networkd", Message: err.Error(),
			})
		}
		return errs
	}

	slog.Debug("preserving NIC naming")
	if err := nicnaming.Apply(mountRoot, staticIPs); err != nil {
		slog.Warn("NIC naming preservation failed", "error", err)
		errs = append(errs, types.BlockError{
			Block: "nic-naming", Message: err.Error(),
		})
	}

	slog.Debug("configuring static IPs")
	if err := staticip.WriteMacToIP(mountRoot, staticIPs); err != nil {
		slog.Warn("writing macToIP failed", "error", err)
		errs = append(errs, types.BlockError{
			Block: "static-ip", Message: err.Error(),
		})
		return errs
	}

	if fbHandler, ok := firstboot.Handlers.Get("systemd"); ok {
		if err := fbHandler.Install(mountRoot, staticip.FirstbootCommands()); err != nil {
			slog.Warn("static IP firstboot install failed", "error", err)
			errs = append(errs, types.BlockError{
				Block: "static-ip/firstboot", Message: err.Error(),
			})
		}
	}

	return errs
}

func scanKernels(mountRoot string) []types.KernelInfo {
	var allKernels []types.KernelInfo
	for _, name := range []string{"rpm", "deb"} {
		scanner, ok := kernel.Scanners.Get(name)
		if !ok {
			continue
		}
		kernels, err := scanner.ScanKernels(mountRoot)
		if err != nil {
			slog.Warn("kernel scan failed", "scanner", name, "error", err)
			continue
		}
		if len(kernels) > 0 {
			slog.Info("found kernels", "count", len(kernels), "scanner", name)
			allKernels = append(allKernels, kernels...)
			break
		}
	}
	return allKernels
}
