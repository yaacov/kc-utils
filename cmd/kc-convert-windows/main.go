//go:build linux

// kc-convert-windows converts a mounted Windows guest for KubeVirt.
// Documentation: docs/kc-convert-windows.md
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/yaacov/kc-utils/pkg/cmd/convert-windows"
	"github.com/yaacov/kc-utils/pkg/common/logger"
	"github.com/yaacov/kc-utils/pkg/common/types"

	// Plugin registrations: registry editor
	_ "github.com/yaacov/kc-utils/pkg/common/registry/hivex"

	// Plugin registrations: driver registrars
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/drivers/plugins/criticaldb"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/drivers/plugins/driverdb"

	// Plugin registrations: driver sources
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/driversource/plugins/directory"

	// Plugin registrations: Windows version handlers
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/version"

	// hypervisor offline remove (block 4)
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor/plugins/remove/awspv"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor/plugins/remove/ec2"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor/plugins/remove/ec2launch"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor/plugins/remove/nutanix"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor/plugins/remove/virtualbox"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor/plugins/remove/vmware"

	// hypervisor service disable (block 8)
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor/plugins/services/nutanix"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor/plugins/services/virtualbox"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor/plugins/services/vmware"

	// Plugin registrations: UEFI editors
	_ "github.com/yaacov/kc-utils/pkg/common/uefi/plugins/bcdeditor"

	// Plugin registrations: firstboot contributors
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/diskonliner"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/multipleips"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/pnputil"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/qemuga"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/routecleanup"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/signal"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/staticipfb"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/vmwarecleanup"
)

func main() {
	prepareFile := flag.String("prepare-data", "", "prepare output JSON")
	outputFile := flag.String("output", "convert-out.json", "output JSON file")
	mountRoot := flag.String("mount-root", "/tmp/kc-guest", "guest mount root")
	offline := flag.Bool("offline", false, "skip network-dependent operations")
	useGuestfs := flag.Bool("guestfs", false, "use libguestfs (FUSE) instead of privileged mount syscalls")
	logLevel := flag.String("log-level", "info", "log level (debug, info, warn, error)")
	flag.Parse()

	logger.Init(*logLevel)

	if *prepareFile == "" {
		fmt.Fprintln(os.Stderr, "error: --prepare-data is required")
		flag.Usage()
		os.Exit(1)
	}

	data, err := os.ReadFile(*prepareFile)
	if err != nil {
		slog.Error("reading prepare data", "error", err)
		os.Exit(1)
	}

	var prepareData types.PrepareOutput
	if err := json.Unmarshal(data, &prepareData); err != nil {
		slog.Error("parsing prepare JSON", "error", err)
		os.Exit(1)
	}

	cfg := &convertwindows.Config{
		PrepareData: prepareData,
		MountRoot:   *mountRoot,
		OutputPath:  *outputFile,
		Offline:     *offline,
		UseGuestfs:  *useGuestfs,
	}

	if err := convertwindows.Run(cfg); err != nil {
		slog.Error("windows convert pipeline failed", "error", err)
		os.Exit(1)
	}
}
