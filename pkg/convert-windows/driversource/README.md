# driversource -- DriverSource interface and driver collection

Provides the abstraction for locating VirtIO driver files on the conversion host and selecting the correct driver set for a given guest architecture and Windows version. The primary entry point, `CollectDrivers`, queries the registered `DriverSource` (typically a directory-based source backed by the virtio-win RPM tree at `/usr/share/virtio-win/drivers/by-os`) and returns a list of `DriverFile` entries ready for copying into the guest.

Architecture names are normalized between guest inspection output and virtio-win directory conventions (e.g. `x86_64` maps to `amd64`). OS version matching uses a canonical alias system that maps Windows product names and shorthand tokens (e.g. "Windows Server 2022" to "2k22") so the correct by-os subdirectory is found. When multiple directories match, handler-specified preferences and fallbacks guide selection. The qemu-ga MSI is conditionally excluded for legacy Windows versions that do not support the guest agent.

## File layout

| File | Purpose |
|------|---------|
| `driversource.go` | Declares the `DriverSource` interface, `DriverFile` struct, and the plugin registry |
| `arch.go` | Architecture normalization and matching helpers |
| `collect.go` | `CollectDrivers` entry point that queries the directory source and filters results |
| `osdir.go` | `FindBestOSDir` / `FindBestOSDirWithPrefs` for selecting the best virtio-win OS subdirectory |
| `osversion.go` | `MatchOSVersion` and `CanonicalOSVersions` alias mapping for Windows version strings |

## Key exports

| Symbol | Role |
|--------|------|
| `DriverSource` | Interface with `Available()` and `FindDrivers()` for locating driver files |
| `DriverFile` | Struct holding a driver's name, source path, INF path, and architecture |
| `Sources` | Plugin registry of `DriverSource` implementations keyed by name |
| `CollectDrivers` | Queries the "directory" source, applies guest-agent filtering, and returns driver files |
| `NormalizeArch` | Maps guest arch names (x86_64, i386, aarch64) to virtio-win directory names (amd64, x86, arm64) |
| `ArchMatches` | Reports whether a directory arch string matches a guest arch after normalization |
| `ArchSearchNames` | Returns all directory name variants to try for a given guest architecture |
| `FindBestOSDir` | Selects the best virtio-win by-os subdirectory for a guest OS version |
| `FindBestOSDirWithPrefs` | Like `FindBestOSDir` but also considers handler preference and fallback lists |
| `MatchOSVersion` | Reports whether a virtio-win directory name matches a requested OS version via canonical aliases |
| `CanonicalOSVersions` | Maps a Windows product name or directory token to its set of canonical aliases |
| `NormalizeOSProductName` | Cleans registry product name strings for case-insensitive substring matching |
