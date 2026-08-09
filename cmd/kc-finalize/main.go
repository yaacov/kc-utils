//go:build linux

// kc-finalize unmounts guest filesystems and writes target metadata.
// Documentation: docs/apps/kc-finalize.md
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/yaacov/kc-utils/pkg/cmd/finalize"
	"github.com/yaacov/kc-utils/pkg/common/logger"
	"github.com/yaacov/kc-utils/pkg/common/types"

	// Plugin registrations: filesystem trimmer
	_ "github.com/yaacov/kc-utils/pkg/finalize/fstrim/plugins/default"

	// Plugin registrations: customizers
	_ "github.com/yaacov/kc-utils/pkg/finalize/customize/plugins/dynamicscriptslinux"
	_ "github.com/yaacov/kc-utils/pkg/finalize/customize/plugins/dynamicscriptswindows"
	_ "github.com/yaacov/kc-utils/pkg/finalize/customize/plugins/native"

	// firstboot handler (used by dynamicscriptslinux customizer)
	_ "github.com/yaacov/kc-utils/pkg/common/firstboot/plugins/systemd"
)

func main() {
	inputFile := flag.String("input", "", "pipeline JSON file")
	outputFile := flag.String("output", "target-meta.json", "output JSON file")
	mountRoot := flag.String("mount-root", "/tmp/kc-guest", "guest mount root")
	useGuestfs := flag.Bool("guestfs", false, "use libguestfs appliance instead of privileged mount syscalls")
	teardownOnly := flag.Bool("teardown-only", false, "reclaim orphaned guest resources without Sync or metadata writes")
	logLevel := flag.String("log-level", "info", "log level (debug, info, warn, error)")
	flag.Parse()

	logger.Init(*logLevel)

	if *teardownOnly {
		cfg := &finalize.Config{
			MountRoot:  *mountRoot,
			UseGuestfs: *useGuestfs,
		}
		if *inputFile != "" {
			data, err := os.ReadFile(*inputFile)
			if err != nil {
				slog.Error("reading input", "error", err)
				os.Exit(1)
			}
			var pipeline types.PipelineData
			if err := json.Unmarshal(data, &pipeline); err != nil {
				slog.Error("parsing pipeline JSON", "error", err)
				os.Exit(1)
			}
			cfg.Pipeline = &pipeline
		}
		if err := finalize.TeardownOnly(cfg); err != nil {
			slog.Error("teardown-only failed", "error", err)
			os.Exit(1)
		}
		return
	}

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

	var pipeline types.PipelineData
	if err := json.Unmarshal(data, &pipeline); err != nil {
		slog.Error("parsing pipeline JSON", "error", err)
		os.Exit(1)
	}
	if pipeline.Prepare == nil || pipeline.Convert == nil {
		slog.Error("pipeline JSON missing 'prepare' and/or 'convert' sections")
		os.Exit(1)
	}

	cfg := &finalize.Config{
		Pipeline:   &pipeline,
		MountRoot:  *mountRoot,
		OutputPath: *outputFile,
		UseGuestfs: *useGuestfs,
	}

	if err := finalize.Run(cfg); err != nil {
		slog.Error("finalize pipeline failed", "error", err)
		os.Exit(1)
	}
}
