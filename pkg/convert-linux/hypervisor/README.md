# hypervisor -- source hypervisor cleanup interface

Defines the `LinuxCleanup` interface for removing source-hypervisor artifacts (tools, drivers, services) from Linux guests during conversion. Concrete implementations register themselves for specific hypervisors (e.g. VMware, Hyper-V) via the plugin registry.

Each `LinuxCleanup` implementation provides `Detect` to check whether the guest has artifacts from that hypervisor and `Cleanup` to remove them. The package also provides two shared helper functions: `DisableSystemdUnit` masks a systemd unit by removing its wants symlinks and symlinking it to `/dev/null`, and `RemovePaths` deletes a list of file paths from the guest filesystem.

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
| `DisableSystemdUnit` | Removes wants symlinks and masks a systemd unit under the guest root |
| `RemovePaths` | Removes a list of file/directory paths from the guest filesystem |
