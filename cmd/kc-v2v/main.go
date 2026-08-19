//go:build unix

// kc-v2v orchestrates prepare → convert → finalize for Forklift migrations.
// Documentation: docs/apps/kc-v2v.md
package main

import (
	"fmt"
	"os"

	v2v "github.com/yaacov/kc-utils/pkg/cmd/v2v"
	"github.com/yaacov/kc-utils/pkg/common/logger"
	"github.com/yaacov/kc-utils/pkg/v2v/env"

	// Backend plugins (shared listener + failure cleanup)
	_ "github.com/yaacov/kc-utils/pkg/backend/plugins/direct"
	_ "github.com/yaacov/kc-utils/pkg/backend/plugins/guestfs"
	_ "github.com/yaacov/kc-utils/pkg/backend/plugins/qemu"
)

func main() {
	logger.Init("info")

	cfg, err := env.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load config:", err)
		os.Exit(1)
	}
	logger.Init(cfg.LogLevel)

	if err := env.LinkCertificates(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "failed to link certificates:", err)
		os.Exit(1)
	}
	if err := env.EnsureWorkdir(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "failed to create workdir:", err)
		os.Exit(1)
	}

	if err := v2v.Run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
