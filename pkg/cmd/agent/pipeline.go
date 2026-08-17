//go:build linux

// Package agent is the kc-agent orchestrator. It bootstraps the appliance
// pid-1 environment, opens the virtio-serial agent port, and serves primitive
// runtime RPCs until the host closes the connection. All domain logic lives
// host-side in pkg/guest/core; this binary is a thin generic runtime.
package agent

import (
	"fmt"
	"os"

	"github.com/yaacov/kc-utils/pkg/agent"
)

// Run bootstraps the appliance and serves the agent until EOF.
func Run() error {
	port, err := agent.Bootstrap()
	if err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	defer port.Close()
	go func() {
		if err := agent.ServeShell(); err != nil {
			fmt.Fprintln(os.Stderr, "kc-agent shell:", err)
		}
	}()
	if err := agent.New().Serve(port); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}
