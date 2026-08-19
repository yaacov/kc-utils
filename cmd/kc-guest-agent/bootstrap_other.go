//go:build !linux

package main

import "errors"

// defaultPort is only meaningful inside the Linux appliance; on other GOOS the
// agent compiles but refuses to run.
const defaultPort = ""

func run(string, bool) error {
	return errors.New("kc-guest-agent runs only inside the Linux appliance")
}
