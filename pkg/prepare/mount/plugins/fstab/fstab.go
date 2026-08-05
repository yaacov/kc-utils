//go:build linux

package fstab

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	cfgfstab "github.com/yaacov/kc-utils/pkg/common/configedit/fstab"
	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest"
	"github.com/yaacov/kc-utils/pkg/prepare/guest/resolve"
	"github.com/yaacov/kc-utils/pkg/prepare/mount"
)

type LinuxPlanner struct{}

func init() {
	mount.Planners.Register("linux", &LinuxPlanner{})
}

func (p *LinuxPlanner) Matches(inspect *types.InspectData) bool {
	return inspect != nil && inspect.Type == "linux"
}

func (p *LinuxPlanner) Plan(ctx *mount.PlanContext) ([]mount.MountSpec, error) {
	ft := ctx.Root.FSType
	if ft == "" {
		var err error
		ft, err = ctx.DetectFS(ctx.Root.DevicePath)
		if err != nil {
			return nil, err
		}
	}
	return []mount.MountSpec{{
		DevicePath: ctx.Root.DevicePath,
		GuestMP:    "/",
		FSType:     ft,
	}}, nil
}

func (p *LinuxPlanner) Expand(ctx *mount.PlanContext, guestRootHost string) ([]mount.MountSpec, error) {
	fstabPath := filepath.Join(guestRootHost, "etc", "fstab")
	data, err := guest.FileRead(fstabPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	catalog, err := resolve.NewCatalog(ctx.Guest, ctx.AllDevices)
	if err != nil {
		return nil, err
	}

	rootDev := ctx.Root.DevicePath
	file := cfgfstab.Parse(string(data))
	var specs []mount.MountSpec
	seen := map[string]bool{"/": true}

	for _, e := range file.DeviceEntries() {
		if skipMountPoint(e.MountPoint, e.FSType) {
			continue
		}
		if seen[e.MountPoint] {
			continue
		}
		dev, err := catalog.Resolve(e.Device)
		if err != nil {
			continue
		}
		if dev == rootDev {
			seen[e.MountPoint] = true
			continue
		}
		ft := e.FSType
		if ft == "" || ft == "auto" {
			ft, err = ctx.DetectFS(dev)
			if err != nil {
				continue
			}
		}
		specs = append(specs, mount.MountSpec{
			DevicePath: dev,
			GuestMP:    e.MountPoint,
			FSType:     ft,
		})
		seen[e.MountPoint] = true
	}

	sort.Slice(specs, func(i, j int) bool {
		return len(specs[i].GuestMP) < len(specs[j].GuestMP)
	})
	return specs, nil
}

func skipMountPoint(mp, fsType string) bool {
	if mp == "" || mp == "none" || mp == "swap" {
		return true
	}
	switch fsType {
	case "swap", "proc", "sysfs", "devpts", "devtmpfs", "tmpfs", "cgroup", "pstore", "bpf", "tracefs", "debugfs", "securityfs", "configfs", "fusectl", "mqueue", "hugetlbfs":
		return true
	}
	if strings.HasPrefix(mp, "/proc") || strings.HasPrefix(mp, "/sys") || strings.HasPrefix(mp, "/dev") {
		return true
	}
	return false
}
