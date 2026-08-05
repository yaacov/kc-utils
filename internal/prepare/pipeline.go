//go:build linux

package prepare

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest"
	"github.com/yaacov/kc-utils/pkg/prepare/converter"
	"github.com/yaacov/kc-utils/pkg/prepare/firmware"
	"github.com/yaacov/kc-utils/pkg/prepare/guest/resolve"
	"github.com/yaacov/kc-utils/pkg/prepare/inspect"
	"github.com/yaacov/kc-utils/pkg/prepare/mount"
	"github.com/yaacov/kc-utils/pkg/prepare/root"
	"github.com/yaacov/kc-utils/pkg/prepare/validate"
)

// Config holds prepare pipeline configuration.
type Config struct {
	Input      types.PrepareInput
	MountRoot  string
	OutputPath string
	UseGuestfs bool
}

// Run executes the prepare pipeline.
func Run(cfg *Config) error {
	slog.Debug("starting pipeline")

	output := &types.PrepareOutput{
		Status: "running",
		Source: cfg.Input.Source,
	}

	if err := validate.Input(len(cfg.Input.Disks), cfg.MountRoot); err != nil {
		return err
	}

	output.MountRoot = cfg.MountRoot
	output.Options = cfg.Input.Options
	for _, d := range cfg.Input.Disks {
		output.Disks = append(output.Disks, types.DiskInfo{
			Path:   d.Path,
			Format: d.Format,
		})
	}

	mode := guest.ModeFromBool(cfg.UseGuestfs)
	slog.Info("opening guest disks", "backend", mode.String(), "disks", len(cfg.Input.Disks), "mountRoot", cfg.MountRoot)

	g, err := guest.Open(cfg.Input.Disks, cfg.MountRoot, mode)
	if err != nil {
		return fmt.Errorf("opening guest disks: %w", err)
	}
	guest.SetActive(g)
	// Leave mounts for convert; only Release process-local probe state.
	defer func() {
		guest.ClearActive()
		if rerr := g.Release(); rerr != nil {
			slog.Warn("guest release failed", "error", rerr)
		}
	}()

	output.Disks = g.DiskInfos()
	lvPaths := g.LVPaths()
	var allPartDevices []string
	for _, d := range output.Disks {
		for _, p := range d.Partitions {
			allPartDevices = append(allPartDevices, p.DevicePath)
		}
	}
	slog.Info("guest disk layout",
		"disks", len(output.Disks),
		"partitions", len(allPartDevices),
		"lvs", len(lvPaths),
	)

	candidateDevices := append([]string{}, allPartDevices...)
	candidateDevices = append(candidateDevices, lvPaths...)

	decryptDisks(g, cfg.Input.LUKS, candidateDevices)

	for _, d := range output.Disks {
		for _, p := range d.Partitions {
			if err := g.FSCheck(p.DevicePath, p.FSType); err != nil {
				slog.Warn("fscheck failed", "device", p.DevicePath, "error", err)
			}
		}
	}

	output.Firmware = firmware.Detect(output.Disks)
	slog.Info("detected firmware",
		"type", output.Firmware.Type,
		"espDevices", output.Firmware.ESPDevices,
	)

	candidates, err := root.Discover(g, output.Disks, lvPaths)
	if err != nil {
		return err
	}
	output.RootCandidates = candidates
	slog.Info("root candidates", "count", len(candidates), "selector", cfg.Input.Options.Root)
	for i := range candidates {
		c := &candidates[i]
		slog.Info("root candidate",
			"index", i+1,
			"device", c.DevicePath,
			"product", c.ProductName,
			"os", c.Inspect.Type,
			"distro", c.Inspect.Distro,
		)
	}

	chosen, err := root.Select(candidates, cfg.Input.Options.Root)
	if err != nil {
		var mb *root.MultiBootError
		if errors.As(err, &mb) {
			writeMultibootError(cfg.OutputPath, output, mb)
		}
		return err
	}
	output.RootDevice = chosen.DevicePath
	slog.Info("selected root",
		"device", chosen.DevicePath,
		"product", chosen.ProductName,
		"selector", cfg.Input.Options.Root,
		"os", chosen.Inspect.Type,
	)

	if err := planAndMount(g, cfg, &chosen, allPartDevices, lvPaths, output); err != nil {
		return err
	}

	output.Firmware = firmware.Detect(output.Disks)
	slog.Info("firmware after mount",
		"type", output.Firmware.Type,
		"espDevices", output.Firmware.ESPDevices,
	)

	if err := inspect.CheckFreeSpace(cfg.MountRoot); err != nil {
		slog.Warn("free space check", "error", err)
	}

	inspectData, err := inspect.InspectGuest(cfg.MountRoot)
	if err != nil {
		slog.Warn("inspection failed", "error", err)
		inspectData = &chosen.Inspect
	}
	output.Inspect = *inspectData
	if output.Inspect.Arch == "" {
		output.Inspect.Arch = inspect.DetectArch(cfg.MountRoot)
	}
	if output.Inspect.Type == "windows" {
		if winInspect, winErr := inspect.InspectWindowsMetadata(cfg.MountRoot); winErr != nil {
			slog.Warn("windows registry inspection failed", "error", winErr)
			output.InspectWindows = winInspect
		} else {
			output.InspectWindows = winInspect
		}
	}

	output.BootDevice = inspect.Detect(cfg.MountRoot, output.Disks)
	output.FreeSpace = inspect.Record(cfg.MountRoot)

	selector, ok := converter.Selectors.Get("default")
	switch {
	case ok:
		conv, err := selector.Select(inspectData)
		if err != nil {
			return fmt.Errorf("selecting converter: %w", err)
		}
		output.Converter = conv
	case inspectData.Type == "windows":
		output.Converter = "kc-convert-windows"
	default:
		output.Converter = "kc-convert-linux"
	}

	output.Status = "complete"

	if err := types.WriteJSON(cfg.OutputPath, output); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	slog.Info("pipeline complete")
	return nil
}

func writeMultibootError(outputPath string, output *types.PrepareOutput, mb *root.MultiBootError) {
	output.Status = "error"
	output.Error = mb.Error()
	output.RootCandidates = mb.Candidates
	_ = types.WriteJSON(outputPath, output)
}

func decryptDisks(g *guest.Guest, luks *types.LUKSSpec, candidateDevices []string) {
	if luks != nil && len(luks.KeyFiles) > 0 {
		for device, keyFile := range luks.KeyFiles {
			if device == "" || device == "all" || strings.HasPrefix(device, "all-") {
				for _, d := range candidateDevices {
					mapper := "v2v-luks-keyfile-" + sanitizeMapper(d)
					if _, err := g.Decrypt(d, keyFile, mapper); err != nil {
						slog.Debug("LUKS keyfile try failed", "device", d, "error", err)
						continue
					}
					slog.Info("LUKS decrypted with keyfile", "device", d)
				}
				continue
			}
			mapper := "v2v-luks-keyfile-" + sanitizeMapper(device)
			if _, err := g.Decrypt(device, keyFile, mapper); err != nil {
				slog.Warn("LUKS decrypt failed", "device", device, "error", err)
			}
		}
	}

	if luks != nil && luks.Clevis {
		for _, device := range candidateDevices {
			if _, err := g.UnlockClevis(device, "v2v-luks-clevis"); err != nil {
				slog.Debug("clevis decrypt skipped", "device", device, "error", err)
			}
		}
	}
}

func planAndMount(g *guest.Guest, cfg *Config, chosen *types.RootCandidate, allPartDevices, lvPaths []string, output *types.PrepareOutput) error {
	allDevices := resolve.AllDevices(allPartDevices, lvPaths)
	planCtx := &mount.PlanContext{
		MountRoot:  cfg.MountRoot,
		Root:       *chosen,
		Disks:      output.Disks,
		Firmware:   output.Firmware,
		LVPaths:    lvPaths,
		AllDevices: allDevices,
		Guest:      g,
	}

	plannerName, planner, initialSpecs, err := mount.Plan(planCtx)
	if err != nil {
		return fmt.Errorf("mount plan: %w", err)
	}
	if planner == nil {
		return fmt.Errorf("no mount planner for OS type %q", chosen.Inspect.Type)
	}
	slog.Info("mount planner selected", "planner", plannerName, "initialMounts", len(initialSpecs))

	if err := mount.Apply(g, initialSpecs, output.Disks); err != nil {
		return fmt.Errorf("mount root: %w", err)
	}

	extraSpecs, err := planner.Expand(planCtx, cfg.MountRoot)
	if err != nil {
		return fmt.Errorf("expand mount plan: %w", err)
	}
	if len(extraSpecs) > 0 {
		slog.Info("expanding mount plan", "extraMounts", len(extraSpecs))
		if err := mount.Apply(g, extraSpecs, output.Disks); err != nil {
			return fmt.Errorf("mount filesystems: %w", err)
		}
	}
	return nil
}

func sanitizeMapper(device string) string {
	base := filepath.Base(device)
	base = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, base)
	if base == "" {
		return "vol"
	}
	return base
}
