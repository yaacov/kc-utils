# version -- VersionHandler interface and Windows version classification

Classifies Windows guests by OS version and drives all version-specific conversion behavior through the `VersionHandler` interface. Each handler declares driver directory preferences, firstboot launcher style, static IP script format, disk online strategy, VMware cleanup method, driver registrar type, and whether the NTFS heads fix is needed.

The package ships concrete handlers for Windows 11, 10, 8.1, 8, 7, Vista, Server 2008/2008 R2, Server 2003, and XP, plus a fallback `winunknown` handler. Handlers are registered in priority order via `init()` (most specific first). `Classify` iterates the registered handlers and returns the first one whose `Matches` method accepts the guest's `InspectData` (major/minor version numbers and product name). Product name matching uses `NormalizeProductName` to strip null bytes, trademark symbols, and extra whitespace before case-insensitive comparison. The `CollectGuestAgentMSI` helper excludes the qemu-ga MSI for legacy versions (XP, 2003, Vista, Server 2008) that cannot run the guest agent.

## File layout

| File | Purpose |
|------|---------|
| `version.go` | `VersionHandler` interface, kind enums (`LauncherKind`, `StaticIPKind`, etc.), `Classify`, and `Register` |
| `handlers.go` | Concrete handler structs for each Windows version (win11 through winxp) and the winunknown fallback |
| `match.go` | `NormalizeProductName`, `isServerProduct`, and `usesWin11DriverSet` helper functions |
| `register.go` | `init()` function that registers all handlers in classification priority order |
| `guestagent.go` | `CollectGuestAgentMSI` and the legacy-version exclusion list |

## Key exports

| Symbol | Role |
|--------|------|
| `VersionHandler` | Interface that drives all version-specific conversion decisions |
| `LauncherKind` | Enum selecting the firstboot.bat template: `LauncherModern`, `LauncherPSV1`, or `LauncherBatOnly` |
| `StaticIPKind` | Enum selecting static IP script format: `StaticIPNetCmdlet`, `StaticIPRegistry`, or `StaticIPWMINetsh` |
| `DiskOnlineKind` | Enum selecting disk online mode: `DiskOnlineGetDisk`, `DiskOnlineWMIDiskpart`, or `DiskOnlineSkip` |
| `VMwareCleanupKind` | Enum selecting VMware cleanup mode: `VMwareCleanupPNP`, `VMwareCleanupDevconBat`, or `VMwareCleanupSkip` |
| `Handlers` | Global plugin registry of `VersionHandler` implementations |
| `Register` | Adds a handler to the registry and records classification order |
| `Classify` | Returns the first matching handler for a guest's `InspectData`, or `winunknown` |
| `NormalizeProductName` | Cleans registry product names for case-insensitive substring matching |
| `CollectGuestAgentMSI` | Reports whether qemu-ga MSIs should be included for a given handler name |
