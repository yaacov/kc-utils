//go:build unix

// kc-agent-sh attaches an interactive bash PTY to a running QEMU appliance.
// Documentation: docs/apps/kc-agent-sh.md
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/yaacov/kc-utils/pkg/cmd/agentsh"
)

func main() {
	sock := flag.String("sock", "", "shell Unix socket (default: $KC_AGENT_SOCK sibling shell.sock)")
	chroot := flag.String("chroot", "", "chroot into a path already mounted in the appliance")
	flag.Parse()

	cfg := agentsh.Config{
		Sock:   *sock,
		Chroot: *chroot,
		Argv:   flag.Args(),
	}
	if err := agentsh.Run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "kc-agent-sh:", err)
		os.Exit(1)
	}
}
