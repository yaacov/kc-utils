package iso

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/convert-windows/driversource"
)

type ISOSource struct {
	ISOPath    string
	extractDir string
}

func init() {
	driversource.Sources.Register("iso", &ISOSource{})
}

func (s *ISOSource) Available() bool {
	if s.ISOPath == "" {
		s.ISOPath = "/usr/share/virtio-win/virtio-win.iso"
	}
	_, err := os.Stat(s.ISOPath)
	return err == nil
}

// Cleanup removes the extracted ISO tree kept alive for drivers.Copy.
func (s *ISOSource) Cleanup() {
	if s.extractDir == "" {
		return
	}
	if err := os.RemoveAll(s.extractDir); err != nil {
		slog.Warn("iso extract cleanup failed", "path", s.extractDir, "error", err)
	}
	s.extractDir = ""
}

func (s *ISOSource) FindDrivers(arch, osVersion string) ([]driversource.DriverFile, error) {
	// Drop any previous extract before creating a new one.
	s.Cleanup()

	tmpDir, err := os.MkdirTemp("", "virtio-iso-")
	if err != nil {
		return nil, fmt.Errorf("iso: failed to create temp dir: %w", err)
	}

	// Host-side ISO access for virtio-win drivers — not guest-disk I/O.
	// Userspace extract only: conversion pods are unprivileged (no /dev/loop).
	if err := extractVirtioISO(s.ISOPath, tmpDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, err
	}
	s.extractDir = tmpDir

	return collectDrivers(tmpDir, arch, osVersion)
}

func extractVirtioISO(isoPath, dest string) error {
	cmd := exec.Command("bsdtar", "--no-xattrs", "-xf", isoPath, "-C", dest)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("iso: failed to extract %s with bsdtar: %w: %s",
			isoPath, err, strings.TrimSpace(string(out)))
	}
	slog.Info("virtio-win ISO extracted", "path", isoPath, "dest", dest)
	return nil
}

func collectDrivers(root, arch, osVersion string) ([]driversource.DriverFile, error) {
	// virtio-win ISO layout: <driverName>/<osVersion>/<arch>/ with .inf/.sys/.cat.
	// qemu-ga MSI packages typically sit at the ISO root.
	var drivers []driversource.DriverFile
	normArch := driversource.NormalizeArch(arch)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		lowerName := strings.ToLower(info.Name())

		if strings.HasSuffix(lowerName, ".msi") && strings.Contains(lowerName, "qemu-ga") {
			if !msiMatchesArch(lowerName, normArch) {
				return nil
			}
			drivers = append(drivers, driversource.DriverFile{
				Name:    "qemu-ga",
				SrcPath: filepath.Dir(path),
				InfPath: path,
				Arch:    arch,
			})
			return nil
		}

		if !strings.HasSuffix(lowerName, ".inf") {
			return nil
		}

		dir := filepath.Dir(path)
		relDir, _ := filepath.Rel(root, dir)
		parts := strings.Split(relDir, string(filepath.Separator))

		// Expect at least <driverName>/<osVersion>/<arch>.
		if len(parts) < 3 {
			return nil
		}

		dirArch := strings.ToLower(parts[len(parts)-1])
		dirOSVer := strings.ToLower(parts[len(parts)-2])

		if !driversource.ArchMatches(dirArch, arch) {
			return nil
		}

		if !driversource.MatchOSVersion(dirOSVer, osVersion) {
			return nil
		}

		drivers = append(drivers, driversource.DriverFile{
			Name:    strings.TrimSuffix(info.Name(), filepath.Ext(info.Name())),
			SrcPath: dir,
			InfPath: path,
			Arch:    arch,
		})
		return nil
	})

	slog.Info("found drivers", "count", len(drivers), "arch", arch, "osVersion", osVersion)
	return drivers, err
}

func msiMatchesArch(msiName, normArch string) bool {
	switch normArch {
	case "amd64":
		return strings.Contains(msiName, "x64") ||
			strings.Contains(msiName, "x86_64") ||
			strings.Contains(msiName, "amd64") ||
			(!strings.Contains(msiName, "x86") && !strings.Contains(msiName, "i386"))
	case "x86":
		return strings.Contains(msiName, "x86") || strings.Contains(msiName, "i386")
	default:
		return true
	}
}
