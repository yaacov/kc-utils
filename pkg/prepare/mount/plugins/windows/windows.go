//go:build unix

package windows

import (
	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/prepare/mount"
)

type Planner struct{}

func init() {
	mount.Planners.Register("windows", &Planner{})
}

func (p *Planner) Matches(inspect *types.InspectData) bool {
	return inspect != nil && inspect.Type == "windows"
}

func (p *Planner) Plan(ctx *mount.PlanContext) ([]mount.MountSpec, error) {
	ft := ctx.Root.FSType
	if ft == "" {
		var err error
		ft, err = ctx.DetectFS(ctx.Root.DevicePath)
		if err != nil {
			return nil, err
		}
	}
	specs := []mount.MountSpec{{
		DevicePath: ctx.Root.DevicePath,
		GuestMP:    "/",
		FSType:     ft,
	}}

	esp := findESP(ctx)
	if esp != "" {
		espType, err := ctx.DetectFS(esp)
		if err == nil && espType == "vfat" {
			specs = append(specs, mount.MountSpec{
				DevicePath: esp,
				GuestMP:    "/boot/efi",
				FSType:     espType,
			})
		}
	}
	return specs, nil
}

func (p *Planner) Expand(_ *mount.PlanContext, _ string) ([]mount.MountSpec, error) {
	return nil, nil
}

func findESP(ctx *mount.PlanContext) string {
	if len(ctx.Firmware.ESPDevices) > 0 {
		return ctx.Firmware.ESPDevices[0]
	}
	di := ctx.Root.DiskIndex
	if di < 0 || di >= len(ctx.Disks) {
		return ""
	}
	for _, p := range ctx.Disks[di].Partitions {
		if p.FSType == "vfat" && p.DevicePath != ctx.Root.DevicePath {
			return p.DevicePath
		}
	}
	return ""
}
