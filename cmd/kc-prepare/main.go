//go:build unix

// kc-prepare opens guest disks, inspects the OS, and mounts filesystems.
// Documentation: docs/apps/kc-prepare.md
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yaacov/kc-utils/pkg/cmd/prepare"
	"github.com/yaacov/kc-utils/pkg/common/logger"
	_ "github.com/yaacov/kc-utils/pkg/common/registry/hivex"
	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest/backend"

	// Guest disk backends
	_ "github.com/yaacov/kc-utils/pkg/guest/plugins/direct"
	_ "github.com/yaacov/kc-utils/pkg/guest/plugins/guestfs"
	_ "github.com/yaacov/kc-utils/pkg/guest/plugins/qemu"

	// Plugin registrations: firmware detectors
	_ "github.com/yaacov/kc-utils/pkg/prepare/firmware/plugins/gptesp"

	// Plugin registrations: converter selector
	_ "github.com/yaacov/kc-utils/pkg/prepare/converter/plugins/default"

	// Plugin registrations: root selection
	_ "github.com/yaacov/kc-utils/pkg/prepare/root/plugins/device"
	_ "github.com/yaacov/kc-utils/pkg/prepare/root/plugins/first"
	_ "github.com/yaacov/kc-utils/pkg/prepare/root/plugins/single"

	// Plugin registrations: mount planning
	_ "github.com/yaacov/kc-utils/pkg/prepare/mount/plugins/fstab"
	_ "github.com/yaacov/kc-utils/pkg/prepare/mount/plugins/windows"
)

func main() {
	inputFile := flag.String("input", "", "input JSON file")
	outputFile := flag.String("output", "prepare-out.json", "output JSON file")
	mountRoot := flag.String("mount-root", "/tmp/kc-guest", "guest mount root")
	diskDir := flag.String("disk-dir", "", "auto-discover disk*.img files from this directory (used when input JSON has no disks)")
	backendName := flag.String("backend", "", backend.BackendFlagUsage()+" (required)")
	logLevel := flag.String("log-level", "info", "log level (debug, info, warn, error)")
	flag.Parse()

	mode, err := backend.ParseMode(*backendName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	resolvedBackend := mode.String()

	logger.Init(*logLevel)
	slog.Info("kc-prepare starting",
		"input", *inputFile,
		"output", *outputFile,
		"mountRoot", *mountRoot,
		"diskDir", *diskDir,
		"backend", resolvedBackend,
		"logLevel", *logLevel,
	)

	if *inputFile == "" {
		fmt.Fprintln(os.Stderr, "error: --input is required")
		flag.Usage()
		os.Exit(1)
	}

	data, err := os.ReadFile(*inputFile)
	if err != nil {
		slog.Error("reading input", "error", err)
		os.Exit(1)
	}

	var input types.PrepareInput
	if err := json.Unmarshal(data, &input); err != nil {
		slog.Error("parsing input JSON", "error", err)
		os.Exit(1)
	}

	if len(input.Disks) == 0 {
		dir := *diskDir
		if dir == "" {
			dir = input.DiskDir
		}
		if dir != "" {
			disks, diskErr := discoverDiskImages(dir)
			if diskErr != nil {
				slog.Error("discovering disks", "dir", dir, "error", diskErr)
				os.Exit(1)
			}
			if len(disks) == 0 {
				fmt.Fprintf(os.Stderr, "error: no disk*.img files found in %s\n", dir)
				os.Exit(1)
			}
			input.Disks = disks
			slog.Info("auto-discovered disks from directory", "dir", dir, "count", len(disks))
		}
	}

	slog.Info("kc-prepare input loaded",
		"disks", len(input.Disks),
		"root", input.Options.Root,
		"firmwareHint", input.Source.FirmwareHint,
		"sourceType", input.Source.Type,
	)

	pipeline := &types.PipelineData{Input: &input}

	cfg := &prepare.Config{
		Input:      input,
		Pipeline:   pipeline,
		MountRoot:  *mountRoot,
		OutputPath: *outputFile,
		Backend:    resolvedBackend,
	}

	if err := prepare.Run(cfg); err != nil {
		slog.Error("prepare pipeline failed", "error", err)
		os.Exit(1)
	}
	slog.Info("kc-prepare completed", "output", *outputFile)
}

// discoverDiskImages finds disk*.img files in dir, sorted by numeric suffix.
func discoverDiskImages(dir string) ([]types.DiskSpec, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	type diskEntry struct {
		name string
		num  int
	}
	var found []diskEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "disk") && strings.HasSuffix(name, ".img") {
			numStr := strings.TrimSuffix(strings.TrimPrefix(name, "disk"), ".img")
			n := 0
			if numStr != "" {
				if _, err := fmt.Sscanf(numStr, "%d", &n); err != nil {
					continue
				}
			}
			found = append(found, diskEntry{name: name, num: n})
		}
	}
	sort.Slice(found, func(i, j int) bool {
		return found[i].num < found[j].num
	})
	disks := make([]types.DiskSpec, 0, len(found))
	for _, d := range found {
		disks = append(disks, types.DiskSpec{
			Path:   filepath.Join(dir, d.name),
			Format: "raw",
		})
	}
	return disks, nil
}
