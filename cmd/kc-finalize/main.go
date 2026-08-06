//go:build linux

// kc-finalize unmounts guest filesystems and writes target metadata.
// Documentation: docs/kc-finalize.md
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
	_ "github.com/yaacov/kc-utils/pkg/finalize/customize/plugins/dynamicscripts"
	_ "github.com/yaacov/kc-utils/pkg/finalize/customize/plugins/native"

	// firstboot handler (used by dynamicscripts customizer)
	_ "github.com/yaacov/kc-utils/pkg/common/firstboot/plugins/systemd"
)

func main() {
	prepareFile := flag.String("prepare-data", "", "prepare output JSON")
	convertFile := flag.String("convert-data", "", "converter output JSON")
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
		if *prepareFile != "" {
			prepareRaw, err := os.ReadFile(*prepareFile)
			if err != nil {
				slog.Error("reading prepare data", "error", err)
				os.Exit(1)
			}
			var prepareData types.PrepareOutput
			if err := json.Unmarshal(prepareRaw, &prepareData); err != nil {
				slog.Error("parsing prepare JSON", "error", err)
				os.Exit(1)
			}
			cfg.PrepareData = prepareData
		}
		if err := finalize.TeardownOnly(cfg); err != nil {
			slog.Error("teardown-only failed", "error", err)
			os.Exit(1)
		}
		return
	}

	if *prepareFile == "" || *convertFile == "" {
		fmt.Fprintln(os.Stderr, "error: --prepare-data and --convert-data are required")
		flag.Usage()
		os.Exit(1)
	}

	prepareRaw, err := os.ReadFile(*prepareFile)
	if err != nil {
		slog.Error("reading prepare data", "error", err)
		os.Exit(1)
	}
	var prepareData types.PrepareOutput
	if err := json.Unmarshal(prepareRaw, &prepareData); err != nil {
		slog.Error("parsing prepare JSON", "error", err)
		os.Exit(1)
	}

	convertRaw, err := os.ReadFile(*convertFile)
	if err != nil {
		slog.Error("reading convert data", "error", err)
		os.Exit(1)
	}
	var convertData types.ConverterOutput
	if err := json.Unmarshal(convertRaw, &convertData); err != nil {
		slog.Error("parsing convert JSON", "error", err)
		os.Exit(1)
	}

	cfg := &finalize.Config{
		PrepareData: prepareData,
		ConvertData: convertData,
		MountRoot:   *mountRoot,
		OutputPath:  *outputFile,
		UseGuestfs:  *useGuestfs,
	}

	if err := finalize.Run(cfg); err != nil {
		slog.Error("finalize pipeline failed", "error", err)
		os.Exit(1)
	}
}
