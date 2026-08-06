// kc-copy runs the disk copy stage (invoked by kc-v2v, or standalone).
// Documentation: docs/kc-copy.md
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/logger"
	kccopy "github.com/yaacov/kc-utils/pkg/copy"
	"github.com/yaacov/kc-utils/pkg/v2v/config"
	"github.com/yaacov/kc-utils/pkg/v2v/env"
)

func main() {
	logger.Init("info")

	inputFile := flag.String("input", "", "input JSON file (CopyInput)")
	libvirtURL := flag.String("libvirt-url", os.Getenv(config.EnvLibvirtURL), "vCenter URL (vpx://...)")
	vmName := flag.String("vm-name", os.Getenv(config.EnvVmName), "VM name")
	fingerprint := flag.String("fingerprint", os.Getenv(config.EnvFingerprint), "vCenter SSL thumbprint")
	diskPath := flag.String("disk-path", os.Getenv(config.EnvDiskPath), "comma-separated source vmdk paths to copy")
	workdir := flag.String("work-dir", config.DefaultWorkdir, "working directory")
	copyConcurrency := flag.Int("copy-concurrency", envInt(config.EnvCopyConcurrency, kccopy.DefaultCopyConcurrency), "max parallel disk copies")
	logLevel := flag.String("log-level", "info", "log level (debug, info, warn, error)")
	flag.Parse()

	logger.Init(*logLevel)

	input, err := loadInput(*inputFile, *libvirtURL, *vmName, *fingerprint, *diskPath, *workdir, *copyConcurrency)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := env.LinkCertificates(&env.Config{Source: "vSphere"}); err != nil {
		fmt.Fprintln(os.Stderr, "failed to link certificates:", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(input.Workdir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "failed to create workdir:", err)
		os.Exit(1)
	}

	if err := kccopy.Run(&input); err != nil {
		fmt.Fprintln(os.Stderr, "copy failed:", err)
		os.Exit(1)
	}
}

func loadInput(inputFile, libvirtURL, vmName, fingerprint, diskPath, workdir string, copyConcurrency int) (kccopy.CopyInput, error) {
	if inputFile != "" {
		data, err := os.ReadFile(inputFile)
		if err != nil {
			return kccopy.CopyInput{}, fmt.Errorf("read input: %w", err)
		}
		var input kccopy.CopyInput
		if err := json.Unmarshal(data, &input); err != nil {
			return kccopy.CopyInput{}, fmt.Errorf("parse input JSON: %w", err)
		}
		if input.Workdir == "" {
			input.Workdir = config.DefaultWorkdir
		}
		if input.CopyConcurrency == 0 {
			input.CopyConcurrency = copyConcurrency
		}
		return input, nil
	}

	input := kccopy.CopyInput{
		VCenterURL:      libvirtURL,
		VMName:          vmName,
		Fingerprint:     fingerprint,
		SourceDisks:     splitDiskPath(diskPath),
		Workdir:         workdir,
		CopyConcurrency: copyConcurrency,
	}
	if err := validateInput(&input); err != nil {
		return kccopy.CopyInput{}, err
	}
	return input, nil
}

func validateInput(input *kccopy.CopyInput) error {
	if input.VCenterURL == "" {
		return fmt.Errorf("--libvirt-url is required (or use --input)")
	}
	if input.VMName == "" {
		return fmt.Errorf("--vm-name is required (or use --input)")
	}
	if input.Fingerprint == "" {
		return fmt.Errorf("--fingerprint is required (or use --input)")
	}
	if len(input.SourceDisks) == 0 {
		return fmt.Errorf("--disk-path is required (or use --input with source_disks)")
	}
	return nil
}

func splitDiskPath(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var paths []string
	for _, part := range strings.Split(raw, ",") {
		if path := strings.TrimSpace(part); path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

func envInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return def
}
