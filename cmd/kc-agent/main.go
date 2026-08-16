//go:build linux

package main

import (
	"fmt"
	"os"

	"github.com/yaacov/kc-utils/pkg/guest/qemu/server"
)

func main() {
	os.Exit(run())
}

func run() int {
	port, err := server.Bootstrap()
	if err != nil {
		fmt.Fprintln(os.Stderr, "kc-agent bootstrap:", err)
		return 1
	}
	defer port.Close()
	agent := server.New()
	if err := agent.Serve(port); err != nil {
		fmt.Fprintln(os.Stderr, "kc-agent:", err)
		return 1
	}
	return 0
}
