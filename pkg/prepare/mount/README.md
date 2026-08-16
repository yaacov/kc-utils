# mount -- mount planning and application

Plans which guest filesystems to mount and in what order, then executes the mounts. The planning step is OS-aware: a pluggable `MountPlanner` interface lets different OS types (Linux, Windows) supply their own mount strategies, while the apply step handles ordering and bookkeeping uniformly.

`Plan` iterates over the `Planners` plugin registry to find a planner whose `Matches` method accepts the inspection data for the chosen root. The planner returns an initial set of `MountSpec` entries (device, guest mount point, filesystem type, read-only flag). After the root is mounted, `Expand` can be called to discover additional mounts (e.g., `/boot`, `/boot/efi` from fstab). `Apply` sorts specs by mount-path length (shallowest first) and mounts each one via the guest wrapper, then records the mount point back into the `DiskInfo` partition table for downstream consumers.

## File layout

| File | Purpose |
|------|---------|
| `plan.go` | Defines `MountSpec`, `PlanContext`, `MountPlanner` interface, and `Plan` dispatch |
| `apply.go` | Mounts specs in order, updates partition metadata |

## Key exports

| Symbol | Role |
|--------|------|
| `MountSpec` | Struct describing one filesystem mount (device, mount point, fstype, read-only) |
| `PlanContext` | Carries state needed by planners: mount root, root candidate, disks, firmware, guest |
| `PlanContext.DetectFS` | Returns the filesystem type for a device via the guest wrapper |
| `MountPlanner` | Interface with `Matches`, `Plan`, and `Expand` methods |
| `Planners` | Global plugin registry of `MountPlanner` implementations |
| `Plan` | Selects the first matching planner and returns its name, instance, and initial specs |
| `Apply` | Mounts specs in path-length order and updates disk partition mount points |
