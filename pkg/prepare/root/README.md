# root -- root discovery and selection

Discovers OS root filesystem candidates on guest disks and selects one according to a configurable policy. This is a central step in the prepare pipeline: every subsequent operation (mount planning, inspection, conversion) depends on identifying the correct root partition.

`Discover` iterates over all disk partitions and LVM volumes, skipping swap, LUKS, and unknown filesystem types. For each eligible device it performs a temporary mount and calls `inspect.ProbeRoot` to check whether the filesystem looks like an OS root (e.g., contains `/etc/os-release` or `Windows/System32`). Matching devices are returned as `RootCandidate` structs. `Select` then applies a policy -- `"single"` (exactly one candidate required), `"first"` (pick the first), or an explicit `/dev/` path -- by dispatching to a `RootSelector` from the plugin registry.

## File layout

| File | Purpose |
|------|---------|
| `discover.go` | Scans partitions and LVM volumes for OS root candidates |
| `select.go` | Applies a selection policy via pluggable `RootSelector` implementations |

## Key exports

| Symbol | Role |
|--------|------|
| `Discover` | Probes all partitions and LVM volumes, returns root candidates |
| `RootSelector` | Interface for root selection policies (`Select` method) |
| `Selectors` | Global plugin registry of `RootSelector` implementations |
| `Select` | Dispatches to a registered selector based on the choice string |
| `MultiBootError` | Error type returned when `"single"` policy finds multiple roots |
