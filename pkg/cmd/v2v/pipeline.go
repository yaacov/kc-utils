//go:build linux

package v2v

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	v2vserver "github.com/yaacov/kc-utils/pkg/cmd/v2v/server"
	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest"
	"github.com/yaacov/kc-utils/pkg/prepare/guest/overlay"
	"github.com/yaacov/kc-utils/pkg/v2v/env"
	"github.com/yaacov/kc-utils/pkg/v2v/inspection/xml"
)

// pipelineResult holds outputs from the kc-utils pipeline subprocesses.
type pipelineResult struct {
	Prepare types.PrepareOutput
	Convert types.ConverterOutput
	Target  types.TargetMeta
}

// Run executes the full kc-v2v orchestration for a loaded config.
func Run(cfg *env.Config) error {
	if env.NeedsCopy(cfg) {
		slog.Info("disk copy enabled",
			"inPlace", cfg.IsInPlace,
			"source", cfg.Source,
			"vm", cfg.VmName,
		)
		sources, err := env.ResolveCopySources(cfg)
		if err != nil {
			return fmt.Errorf("resolve copy sources: %w", err)
		}
		slog.Info("resolved copy sources", "count", len(sources), "disks", sources)
		if err := env.ValidateCopySourceCount(sources); err != nil {
			return fmt.Errorf("copy source count: %w", err)
		}
		input := env.BuildCopyInput(cfg, sources)
		inputPath := filepath.Join(cfg.Workdir, "copy-input.json")
		if err := types.WriteJSON(inputPath, input); err != nil {
			return fmt.Errorf("write copy input: %w", err)
		}
		copyBin := filepath.Join(cfg.BinDir, "kc-copy")
		if err := runSubprocess(copyBin, []string{"--input", inputPath, "--log-level", cfg.LogLevel}, nil); err != nil {
			return fmt.Errorf("kc-copy: %w", err)
		}
		cfg.IsInPlace = true
	} else {
		slog.Info("disk copy skipped (in-place)", "inPlace", cfg.IsInPlace, "source", cfg.Source)
	}

	disks, err := env.DiscoverDisks(cfg)
	if err != nil {
		return fmt.Errorf("discover disks: %w", err)
	}

	result, err := runPipeline(cfg, disks)
	if err != nil {
		dumpPartialResults(cfg.Workdir)
		return fmt.Errorf("conversion: %w", err)
	}

	if err := xml.WriteInspectionXML(&result.Target, cfg.InspectionOutputFile); err != nil {
		return fmt.Errorf("write inspection XML: %w", err)
	}
	slog.Info("wrote inspection XML", "path", cfg.InspectionOutputFile)

	if data, err := json.MarshalIndent(result.Target, "", "  "); err == nil {
		slog.Info("pipeline output", "target_meta", string(data))
	}

	v2vserver.SetWarnings(result.Target.Warnings)
	if cfg.IsLocalMigration {
		if err := v2vserver.Start(cfg); err != nil {
			return fmt.Errorf("HTTP server: %w", err)
		}
	}
	return nil
}

func runPipeline(cfg *env.Config, disks []env.DiskInfo) (*pipelineResult, error) {
	if !cfg.OverlayEnabled {
		slog.Info("qcow2 overlay disabled")
		return runPipelineOnce(cfg, disks)
	}
	slog.Info("qcow2 overlay enabled", "disks", len(disks), "workdir", cfg.Workdir)
	overlayDisks := env.ToOverlayDisks(disks)
	var result *pipelineResult
	err := overlay.RunWithOverlay(cfg.Workdir, overlayDisks, func() error {
		var runErr error
		result, runErr = runPipelineOnce(cfg, env.ActiveDiskPaths(overlayDisks))
		return runErr
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func runPipelineOnce(cfg *env.Config, disks []env.DiskInfo) (*pipelineResult, error) {
	source, err := env.FetchSourceMeta(cfg)
	if err != nil {
		return nil, err
	}
	input, err := env.BuildPrepareInput(cfg, disks, source)
	if err != nil {
		return nil, err
	}

	inputPath := filepath.Join(cfg.Workdir, "prepare-input.json")
	prepareOut := filepath.Join(cfg.Workdir, "prepare-out.json")
	convertOut := filepath.Join(cfg.Workdir, "convert-out.json")
	targetMeta := filepath.Join(cfg.Workdir, "target-meta.json")

	if err := types.WriteJSON(inputPath, input); err != nil {
		return nil, err
	}

	return runPipelineOnceBody(cfg, &input, inputPath, prepareOut, convertOut, targetMeta)
}

func runPipelineOnceBody(cfg *env.Config, input *types.PrepareInput, inputPath, prepareOut, convertOut, targetMeta string) (*pipelineResult, error) {
	var pipelineErr error
	guestSetupStarted := false

	var sharedListener *guest.SharedListener
	var stageEnv []string
	if cfg.UseGuestfs {
		listener, err := guest.StartSharedListener()
		if err != nil {
			return nil, fmt.Errorf("guestfish shared listener: %w", err)
		}
		sharedListener = listener
		stageEnv = listener.Env()
	}
	defer func() {
		if sharedListener != nil {
			if err := sharedListener.Close(); err != nil {
				slog.Warn("guestfish shared listener exit failed", "error", err)
			}
		}
	}()

	defer func() {
		if pipelineErr != nil && guestSetupStarted {
			bestEffortGuestCleanup(cfg, prepareOut, stageEnv)
		}
	}()

	fail := func(err error) (*pipelineResult, error) {
		pipelineErr = err
		return nil, err
	}

	prepareBin := filepath.Join(cfg.BinDir, "kc-prepare")
	prepareArgs := []string{
		"--input", inputPath,
		"--output", prepareOut,
		"--mount-root", cfg.MountRoot,
		"--log-level", cfg.LogLevel,
	}
	if cfg.UseGuestfs {
		prepareArgs = append(prepareArgs, "--guestfs")
	}

	guestSetupStarted = true
	if err := runSubprocess(prepareBin, prepareArgs, stageEnv); err != nil {
		return fail(fmt.Errorf("kc-prepare: %w", err))
	}

	if err := ensureSharedListener(sharedListener, &stageEnv, "prepare"); err != nil {
		return fail(err)
	}

	var prepare types.PrepareOutput
	if err := readJSON(prepareOut, &prepare); err != nil {
		return fail(err)
	}
	if prepare.Status == "error" && len(prepare.RootCandidates) > 0 && cfg.RootDisk != "" {
		input.Options.Root = cfg.RootDisk
		if err := types.WriteJSON(inputPath, input); err != nil {
			return fail(err)
		}
		slog.Info("retrying kc-prepare with root selector", "root", cfg.RootDisk)
		if err := runSubprocess(prepareBin, prepareArgs, stageEnv); err != nil {
			return fail(fmt.Errorf("kc-prepare retry: %w", err))
		}
		if err := readJSON(prepareOut, &prepare); err != nil {
			return fail(err)
		}
	}
	if prepare.Status == "error" {
		return fail(fmt.Errorf("kc-prepare failed: %s", prepare.Error))
	}

	converter := prepare.Converter
	if converter == "" {
		if prepare.Inspect.Type == "windows" {
			converter = "kc-convert-windows"
		} else {
			converter = "kc-convert-linux"
		}
	}
	convertBin := filepath.Join(cfg.BinDir, converter)
	convertArgs := []string{
		"--prepare-data", prepareOut,
		"--output", convertOut,
		"--mount-root", cfg.MountRoot,
		"--log-level", cfg.LogLevel,
	}
	if cfg.Offline {
		convertArgs = append(convertArgs, "--offline")
	}
	if cfg.UseGuestfs {
		convertArgs = append(convertArgs, "--guestfs")
	}
	if err := runSubprocess(convertBin, convertArgs, stageEnv); err != nil {
		return fail(fmt.Errorf("%s: %w", converter, err))
	}

	if err := ensureSharedListener(sharedListener, &stageEnv, "conversion"); err != nil {
		return fail(err)
	}

	finalizeBin := filepath.Join(cfg.BinDir, "kc-finalize")
	finalizeArgs := []string{
		"--prepare-data", prepareOut,
		"--convert-data", convertOut,
		"--output", targetMeta,
		"--mount-root", cfg.MountRoot,
		"--log-level", cfg.LogLevel,
	}
	if cfg.UseGuestfs {
		finalizeArgs = append(finalizeArgs, "--guestfs")
	}
	if err := runSubprocess(finalizeBin, finalizeArgs, stageEnv); err != nil {
		return fail(fmt.Errorf("kc-finalize: %w", err))
	}

	var convert types.ConverterOutput
	var target types.TargetMeta
	if err := readJSON(convertOut, &convert); err != nil {
		return fail(err)
	}
	if err := readJSON(targetMeta, &target); err != nil {
		return fail(err)
	}
	return &pipelineResult{Prepare: prepare, Convert: convert, Target: target}, nil
}

// teardownOnlyArgs builds kc-finalize --teardown-only arguments.
func teardownOnlyArgs(cfg *env.Config, prepareOut string) []string {
	args := []string{
		"--teardown-only",
		"--mount-root", cfg.MountRoot,
		"--log-level", cfg.LogLevel,
	}
	if _, err := os.Stat(prepareOut); err == nil {
		args = append(args, "--prepare-data", prepareOut)
	}
	if cfg.UseGuestfs {
		args = append(args, "--guestfs")
	}
	return args
}

// bestEffortGuestCleanup reclaims orphaned guest resources after a pipeline
// failure. Cleanup errors are logged and do not replace the original error.
func bestEffortGuestCleanup(cfg *env.Config, prepareOut string, stageEnv []string) {
	finalizeBin := filepath.Join(cfg.BinDir, "kc-finalize")
	args := teardownOnlyArgs(cfg, prepareOut)
	slog.Info("best-effort guest cleanup after failure", "bin", finalizeBin, "args", args)
	if err := runSubprocess(finalizeBin, args, stageEnv); err != nil {
		slog.Warn("guest cleanup failed", "error", err)
	}
}

func runSubprocess(bin string, args []string, extraEnv []string) error {
	slog.Info("exec", "bin", bin, "args", args)
	cmd := exec.Command(bin, args...)
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}
	var buf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &buf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &buf)
	if err := cmd.Run(); err != nil {
		out := strings.TrimSpace(buf.String())
		if out == "" {
			return err
		}
		const maxTail = 8192
		if len(out) > maxTail {
			out = out[len(out)-maxTail:]
		}
		return fmt.Errorf("%w\n--- subprocess output (tail) ---\n%s", err, out)
	}
	return nil
}

func ensureSharedListener(listener *guest.SharedListener, stageEnv *[]string, stage string) error {
	if listener == nil || guest.SharedListenerAlive(listener) {
		return nil
	}
	slog.Warn("guestfish shared listener died, restarting", "after", stage)
	newListener, err := guest.StartSharedListener()
	if err != nil {
		return fmt.Errorf("guestfish restart after %s: %w", stage, err)
	}
	if closeErr := listener.Close(); closeErr != nil {
		slog.Debug("closing dead shared listener", "error", closeErr)
	}
	*listener = *newListener
	*stageEnv = listener.Env()
	return nil
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func dumpPartialResults(workdir string) {
	for _, name := range []string{"convert-out.json", "prepare-out.json", "target-meta.json"} {
		path := filepath.Join(workdir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		slog.Info("partial pipeline output", "file", name, "content", string(data))
	}
}
