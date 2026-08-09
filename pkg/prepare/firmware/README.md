# firmware -- firmware type detection

Determines whether a guest uses BIOS or UEFI firmware by examining the disk partition layout. The result influences mount planning (e.g., whether an EFI System Partition needs to be mounted) and converter behavior.

The `FirmwareDetector` interface allows pluggable detection strategies registered in a global plugin registry. The convenience function `Detect` defaults to BIOS and then consults the `"gpt-esp"` detector if registered; when that detector finds a GPT EFI System Partition it returns UEFI firmware info instead.

## File layout

| File | Purpose |
|------|---------|
| `firmware.go` | Defines the `FirmwareDetector` interface and `Detectors` plugin registry |
| `detect.go` | `Detect` convenience function that defaults to BIOS and queries `"gpt-esp"` |

## Key exports

| Symbol | Role |
|--------|------|
| `FirmwareDetector` | Interface with a `Detect(disks) (*FirmwareInfo, error)` method |
| `Detectors` | Global plugin registry of `FirmwareDetector` implementations |
| `Detect` | Convenience function: returns BIOS by default, UEFI if `"gpt-esp"` detector matches |
