//go:build linux

// kc-prepare opens guest disks, inspects the OS, and mounts filesystems.
// Documentation: docs/kc-prepare.md
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/yaacov/kc-utils/internal/prepare"
	"github.com/yaacov/kc-utils/pkg/common/logger"
	_ "github.com/yaacov/kc-utils/pkg/common/registry/hivex"
	"github.com/yaacov/kc-utils/pkg/common/types"

	// Plugin registrations: firmware detectors
	_ "github.com/yaacov/kc-utils/pkg/prepare/firmware/plugins/gptesp"

	// Plugin registrations: converter selector
	_ "github.com/yaacov/kc-utils/pkg/prepare/converter/plugins/default"

	// Plugin registrations: LUKS decryptors
	_ "github.com/yaacov/kc-utils/pkg/prepare/decryptor/plugins/clevis"
	_ "github.com/yaacov/kc-utils/pkg/prepare/decryptor/plugins/keyfile"

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
	useGuestfs := flag.Bool("guestfs", false, "use libguestfs appliance instead of privileged mount syscalls")
	logLevel := flag.String("log-level", "info", "log level (debug, info, warn, error)")
	flag.Parse()

	logger.Init(*logLevel)
	slog.Info("kc-prepare starting",
		"input", *inputFile,
		"output", *outputFile,
		"mountRoot", *mountRoot,
		"guestfs", *useGuestfs,
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
	slog.Info("kc-prepare input loaded",
		"disks", len(input.Disks),
		"root", input.Options.Root,
		"firmwareHint", input.Source.FirmwareHint,
		"sourceType", input.Source.Type,
	)

	cfg := &prepare.Config{
		Input:      input,
		MountRoot:  *mountRoot,
		OutputPath: *outputFile,
		UseGuestfs: *useGuestfs,
	}

	if err := prepare.Run(cfg); err != nil {
		slog.Error("prepare pipeline failed", "error", err)
		os.Exit(1)
	}
	slog.Info("kc-prepare completed", "output", *outputFile)
}
