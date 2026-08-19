// kc-guest-agent runs inside the minimal qemu appliance and serves the
// primitive operations defined in pkg/qemuagent/proto over a virtio-serial
// port. The host qemu backend composes all conversion logic from these
// primitives.
//
// The agent is only ever executed inside the Linux appliance; on other GOOS it
// compiles (so `go build ./...` works on a dev host) but refuses to run.
// Documentation: docs/architecture/qemu-appliance.md
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	port := flag.String("port", defaultPort, "virtio-serial port device to serve on")
	asInit := flag.Bool("init", false, "act as PID 1: mount core filesystems before serving")
	flag.Parse()

	if err := run(*port, *asInit || os.Getpid() == 1); err != nil {
		fmt.Fprintln(os.Stderr, "kc-guest-agent:", err)
		os.Exit(1)
	}
}
