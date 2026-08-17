//go:build linux

// kc-agent is the in-appliance generic runtime served over virtio-serial.
// Documentation: docs/apps/kc-agent.md
package main

import (
	"fmt"
	"os"

	agent "github.com/yaacov/kc-utils/pkg/cmd/agent"
)

func main() {
	if err := agent.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "kc-agent:", err)
		os.Exit(1)
	}
}
