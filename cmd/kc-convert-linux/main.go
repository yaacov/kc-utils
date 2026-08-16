//go:build linux

// kc-convert-linux converts a mounted Linux guest for KubeVirt.
// Documentation: docs/apps/kc-convert-linux.md
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/yaacov/kc-utils/pkg/cmd/convert-linux"
	"github.com/yaacov/kc-utils/pkg/common/logger"
	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest"

	// Guest disk backends
	_ "github.com/yaacov/kc-utils/pkg/guest/direct"
	_ "github.com/yaacov/kc-utils/pkg/guest/guestfs"

	// Plugin registrations: bootloader handlers
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/bootloader/plugins/bls"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/bootloader/plugins/grub2"

	// Plugin registrations: distro classifiers
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/distro/plugins/debian"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/distro/plugins/rhel"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/distro/plugins/suse"

	// Plugin registrations: kernel inspectors
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/kernel/plugins/deb"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/kernel/plugins/rpm"

	// hypervisor cleanups (block 11)
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor/plugins/citrix"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor/plugins/ec2"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor/plugins/hyperv"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor/plugins/kudzu"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor/plugins/nutanix"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor/plugins/parallels"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor/plugins/virtualbox"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor/plugins/vmware"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor/plugins/xen"

	// Plugin registrations: device remappers
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/remap/plugins/standard"

	// Plugin registrations: UEFI editors
	_ "github.com/yaacov/kc-utils/pkg/common/uefi/plugins/bcdeditor"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/uefi/plugins/grubfallback"

	// Plugin registrations: guest agent + firstboot + package sources
	_ "github.com/yaacov/kc-utils/pkg/common/firstboot/plugins/systemd"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/guestagent/plugins/agent/qemuga"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/guestagent/plugins/packagesource/directory"

	// Plugin registrations: guest network stack handlers
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/network/handlers/default"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/network/handlers/networkd"

	// Plugin registrations: NIC naming preservation
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming/plugins/dhclient"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming/plugins/ifcfg"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming/plugins/netplan"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming/plugins/nm"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming/plugins/nmdhcp"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming/plugins/wicked"
)

func main() {
	inputFile := flag.String("input", "", "pipeline JSON file")
	outputFile := flag.String("output", "convert-out.json", "output JSON file")
	mountRoot := flag.String("mount-root", "/tmp/kc-guest", "guest mount root")
	offline := flag.Bool("offline", false, "skip network-dependent operations (use local packages only)")
	backend := flag.String("backend", "direct", guest.BackendFlagUsage())
	logLevel := flag.String("log-level", "info", "log level (debug, info, warn, error)")
	flag.Parse()

	mode, err := guest.ParseMode(*backend)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	resolvedBackend := mode.String()

	logger.Init(*logLevel)

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
	if pipeline.Prepare == nil {
		slog.Error("pipeline JSON missing 'prepare' section")
		os.Exit(1)
	}

	cfg := &convertlinux.Config{
		PrepareData: *pipeline.Prepare,
		Pipeline:    &pipeline,
		MountRoot:   *mountRoot,
		OutputPath:  *outputFile,
		Offline:     *offline,
		Backend:     resolvedBackend,
	}

	if err := convertlinux.Run(cfg); err != nil {
		slog.Error("linux convert pipeline failed", "error", err)
		os.Exit(1)
	}
}
