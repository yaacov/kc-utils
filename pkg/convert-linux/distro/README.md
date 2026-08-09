# distro -- distribution detection and package format helpers

Defines the `DistroHandler` interface for matching a guest's inspection data to a specific Linux distribution and providing distro-specific defaults (kernel arguments, console device). Also provides two standalone helper functions for mapping a distro ID string to its package format and package manager command.

`DistroHandler` implementations register into a global plugin registry and are matched against `InspectData` at conversion time. `Format` maps distro IDs to either `"rpm"` or `"deb"` (defaulting to `"rpm"` for unrecognized distros). `Name` maps distro IDs to the package manager command family: `"apt"` for Debian/Ubuntu/ALT, `"zypper"` for SUSE variants, and `"dnf"` for everything else.

## Key exports

| Symbol | Role |
|--------|------|
| `DistroHandler` | Interface for matching inspection data and providing distro-specific defaults (Matches, DefaultKernelArgs, DefaultConsole) |
| `Handlers` | Global plugin registry of `DistroHandler` implementations |
| `Format` | Returns the package format (`"rpm"` or `"deb"`) for a distro ID string |
| `Name` | Returns the package manager command (`"apt"`, `"zypper"`, or `"dnf"`) for a distro ID string |
