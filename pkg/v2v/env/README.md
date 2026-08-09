# env -- PrepareInput builder from v2v config

Loads the kc-v2v runtime configuration from environment variables and CLI flags, then builds the `PrepareInput` structure consumed by the prepare pipeline. This package bridges the gap between Forklift's `V2V_*` environment and the internal pipeline types.

`Load` reads all `V2V_*` environment variables and CLI flags into a `Config` struct, resolves CA certificate paths for vSphere TLS, validates copy mode against PVC state, and returns the populated config. `BuildPrepareInput` assembles a `PrepareInput` from the config, discovered disks, and source metadata (static IPs, LUKS spec, disk specs, and prepare options). Supporting functions handle disk discovery, source metadata fetching from vCenter, static IP parsing, LUKS key loading, copy input construction, and certificate symlink management.

## File layout

| File | Purpose |
|------|---------|
| `alias.go` | Re-exports `config.Config` type and all `config.Env*`/`Default*` constants for backward compatibility |
| `build.go` | `BuildPrepareInput`, `SourceName`, `NormalizeSourceType` -- constructs `PrepareInput` from config and disks |
| `copy.go` | `IsVSphereSource`, `NeedsCopy`, `ValidateCopyMode`, `ResolveCopySources`, `ValidateCopySourceCount`, `BuildCopyInput`, `SplitDiskPath` -- disk copy orchestration helpers |
| `disks.go` | `DiskInfo`, `DiscoverDisks`, `ToOverlayDisks`, `ActiveDiskPaths` -- PVC block/filesystem disk discovery |
| `load.go` | `Load`, `LinkCertificates`, `EnsureWorkdir` -- config loading from env/flags, CA symlinks, workdir creation |
| `luks.go` | `BuildLUKSSpec` -- builds LUKS configuration from Clevis flag or key files |
| `sourcemeta.go` | `FetchSourceMeta` -- loads NIC and firmware metadata from env or vCenter inventory |
| `staticip.go` | `ParseStaticIPs` -- parses the `V2V_staticIPs` format into structured `StaticIP` entries |

## Key exports

| Symbol | Role |
|--------|------|
| `Config` | Type alias for `config.Config` (re-exported for convenience) |
| `Load` | Reads `V2V_*` env vars and CLI flags into a `Config`, validates copy mode |
| `BuildPrepareInput` | Assembles `types.PrepareInput` from config, discovered disks, and source spec |
| `DiscoverDisks` | Finds attached block devices and filesystem disk images via glob patterns |
| `DiskInfo` | Struct holding a discovered disk path |
| `FetchSourceMeta` | Loads NIC/firmware metadata from env vars or vCenter for the source VM |
| `ParseStaticIPs` | Parses the underscore/colon-delimited `V2V_staticIPs` string into `[]types.StaticIP` |
| `BuildLUKSSpec` | Builds `*types.LUKSSpec` from Clevis flag or LUKS key directory |
| `IsVSphereSource` | Reports whether the config source is a vSphere migration |
| `NeedsCopy` | Reports whether disk copy should run (true when not in-place) |
| `ValidateCopyMode` | Checks that PVC empty/populated state matches the `V2V_inPlace` flag |
| `ResolveCopySources` | Returns ordered VMDK paths from `V2V_diskPath` or vCenter inventory |
| `BuildCopyInput` | Maps config and source disk paths to a `kccopy.CopyInput` struct |
| `SplitDiskPath` | Splits a comma-separated disk path string into individual paths |
| `LinkCertificates` | Creates the vSphere CA bundle symlink (idempotent) |
| `EnsureWorkdir` | Creates the v2v working directory |
| `NormalizeSourceType` | Canonicalizes source type strings (e.g. `"nutanix-ahv"` to `"nutanix"`) |
| `SourceName` | Returns the VM name from config, defaulting to `"disk"` |
| `ToOverlayDisks` | Converts `[]DiskInfo` to overlay `[]*overlay.Disk` |
| `ActiveDiskPaths` | Extracts current paths from overlay disks back into `[]DiskInfo` |
