# hypervisor -- source hypervisor cleanup interface

Defines the `LinuxCleanup` interface for removing source-hypervisor artifacts (tools, drivers, services) from Linux guests during conversion. Concrete implementations register themselves for specific hypervisors (e.g. VMware, Hyper-V) via the plugin registry.

Each `LinuxCleanup` implementation provides `Detect` to check whether the guest has artifacts from that hypervisor and `Cleanup` to remove them. The package also provides shared helper functions: `DisableSystemdUnit` masks a systemd unit by removing wants symlinks under `multi-user.target`, `default.target`, `sockets.target`, and `graphical.target` (both `/etc/systemd/system` and vendor `/usr/lib/systemd/system` presets), then symlinking the unit to `/dev/null`; and `RemovePaths` deletes a list of file paths from the guest filesystem.

## File layout

| File | Purpose |
|------|---------|
| `hypervisor.go` | `LinuxCleanup` interface and `LinuxCleanups` plugin registry |
| `systemd.go` | `DisableSystemdUnit` and `RemovePaths` helper functions |

## Key exports

| Symbol | Role |
|--------|------|
| `LinuxCleanup` | Interface for detecting and removing hypervisor artifacts (Detect, Cleanup) |
| `LinuxCleanups` | Global plugin registry of `LinuxCleanup` implementations |
| `DisableSystemdUnit` | Removes wants symlinks from standard and vendor preset targets, then masks the unit under the guest root |
| `SystemdUnitMaskPath` | Returns the host path to the mask symlink for a unit |
| `VendorWantsPath` | Returns the host path to the vendor preset wants symlink for a unit |
| `UnitMaskTarget` | Guest-absolute mask target (`/dev/null`) written by `DisableSystemdUnit` |
| `RemovePaths` | Removes a list of file/directory paths from the guest filesystem |
