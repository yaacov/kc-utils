# hypervisor -- source hypervisor cleanup interface

Defines the `LinuxCleanup` interface for removing source-hypervisor artifacts (tools, drivers, services) from Linux guests during conversion. Concrete implementations register themselves for specific hypervisors (e.g. VMware, Hyper-V) via the plugin registry.

Each `LinuxCleanup` implementation provides `Detect` to check whether the guest has artifacts from that hypervisor and `Cleanup` to remove them.

Systemd unit helpers (`DisableSystemdUnit`, `DisableEC2NetHooks`, etc.) live in [`systemd/`](../systemd/) — not in this package.

## File layout

| File | Purpose |
|------|---------|
| `hypervisor.go` | `LinuxCleanup` interface and `LinuxCleanups` plugin registry |

## Key exports

| Symbol | Role |
|--------|------|
| `LinuxCleanup` | Interface for detecting and removing hypervisor artifacts (Detect, Cleanup) |
| `LinuxCleanups` | Global plugin registry of `LinuxCleanup` implementations |
