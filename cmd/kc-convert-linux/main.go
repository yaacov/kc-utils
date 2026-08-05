//go:build linux

// kc-convert-linux converts a mounted Linux guest for KubeVirt.
// Documentation: docs/kc-convert-linux.md
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/yaacov/kc-utils/internal/convert-linux"
	"github.com/yaacov/kc-utils/pkg/common/logger"
	"github.com/yaacov/kc-utils/pkg/common/types"

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

	// Plugin registrations: NIC naming preservation
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming/plugins/dhclient"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming/plugins/ifcfg"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming/plugins/netplan"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming/plugins/nm"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming/plugins/nmdhcp"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming/plugins/wicked"
)

func main() {
	prepareFile := flag.String("prepare-data", "", "prepare output JSON")
	outputFile := flag.String("output", "convert-out.json", "output JSON file")
	mountRoot := flag.String("mount-root", "/tmp/kc-guest", "guest mount root")
	offline := flag.Bool("offline", false, "skip network-dependent operations (use local packages only)")
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

	cfg := &convertlinux.Config{
		PrepareData: prepareData,
		MountRoot:   *mountRoot,
		OutputPath:  *outputFile,
		Offline:     *offline,
		UseGuestfs:  *useGuestfs,
	}

	if err := convertlinux.Run(cfg); err != nil {
		slog.Error("linux convert pipeline failed", "error", err)
		os.Exit(1)
	}
}
