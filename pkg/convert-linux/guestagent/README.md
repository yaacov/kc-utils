# guestagent -- qemu-guest-agent install orchestration

Orchestrates the detection, removal, and installation of the QEMU guest agent inside a converted VM. Handles both offline installation from local package files and network-based installation via a firstboot systemd script, and supports writing static IP configuration for first-boot network setup.

The `Install` function first checks whether `qemu-guest-agent` is already present via registered `GuestAgent` detectors. If not, it searches registered `PackageSource` implementations for a local package matching the guest's distro, arch, and major version. When a local package is found, it is copied into the guest and a firstboot script is generated to install it with `rpm` or `dpkg`. Otherwise (and if not in offline mode), a firstboot script uses the appropriate package manager (`dnf`, `apt`, or `zypper`) to install from the network. Static IP helpers write a MAC-to-IP mapping file and generate firstboot shell commands to configure interfaces via `nmcli` or `ip`.

## File layout

| File | Purpose |
|------|---------|
| `agent.go` | `GuestAgent` interface and `Agents` plugin registry for detecting/removing guest agents |
| `install.go` | `Install` function: local-package or network-based qemu-guest-agent installation via firstboot |
| `packagesource.go` | `PackageSource` interface, `FindRequest`/`PackageFile` types, and `Sources` registry |
| `staticip.go` | Static IP helpers: `MacToIPLine`, `WriteMacToIP`, `FirstbootCommands` |

## Key exports

| Symbol | Role |
|--------|------|
| `GuestAgent` | Interface for detecting and removing a guest agent (Detect, Remove) |
| `Agents` | Global plugin registry of `GuestAgent` implementations |
| `Install` | Installs qemu-guest-agent via local packages or network firstboot script |
| `PackageSource` | Interface for locating local packages on the conversion host |
| `FindRequest` | Struct describing a local package lookup (name, format, arch, distro, major version) |
| `PackageFile` | Struct describing a found package file (name, host path, format, arch, EL tag) |
| `Sources` | Global plugin registry of `PackageSource` implementations |
| `MacToIPLine` | Formats a single static IP entry for the macToIP mapping file |
| `WriteMacToIP` | Writes the macToIP mapping file into the guest filesystem |
| `FirstbootCommands` | Returns shell commands that configure static IPs on first boot |
