# inspect -- OS type detection, architecture, boot device, and free space

Inspects a mounted guest root filesystem to determine the operating system type (Linux or Windows), distribution, version, CPU architecture, boot device layout, and available free space. The results feed into converter selection, mount planning, and pre-conversion validation.

`InspectGuest` is the top-level entry point: it checks for a Windows system directory and dispatches to either `inspectWindows` (which reads the Windows registry for product name and version) or `inspectLinux` (which parses `/etc/os-release`, `/etc/redhat-release`, or `/etc/debian_version`). `DetectArch` infers the guest CPU architecture from kernel module directory names. `Detect` (boot device) locates the boot partition by scanning for GRUB configs and EFI directories, then matching mount points. `ProbeRoot` is the lightweight variant used during root discovery to test whether a mounted path looks like an OS root. `CheckFreeSpace` verifies that `/`, `/boot`, and `/boot/efi` have enough free bytes and inodes for conversion to succeed.

## File layout

| File | Purpose |
|------|---------|
| `inspect.go` | Top-level `InspectGuest` dispatcher and Windows detection |
| `inspect_linux_guest.go` | Linux inspection: parses os-release, redhat-release, debian_version |
| `inspect_windows_guest.go` | Windows inspection: reads registry hives for version and product name |
| `probe.go` | `ProbeRoot` for root discovery; `ProductName` helper |
| `arch.go` | `DetectArch`: infers CPU architecture from kernel module paths |
| `bootdevice.go` | `Detect`: identifies boot device and bootloader type |
| `freespace.go` | `Record` and `CheckFreeSpace`: free space and inode checks |

## Key exports

| Symbol | Role |
|--------|------|
| `InspectGuest` | Detects OS type and returns `InspectData` for a mounted guest root |
| `ProbeRoot` | Lightweight check whether a mount path is an OS root; used during discovery |
| `ProductName` | Returns a human-readable OS name from `InspectData` |
| `DetectArch` | Infers guest CPU architecture from kernel module directory names |
| `Detect` | Identifies boot device index, partition, and bootloader type |
| `Record` | Returns free space stats for candidate mount paths |
| `CheckFreeSpace` | Validates that mounted filesystems have enough space for conversion |
| `InspectWindowsMetadata` | Reads Windows registry hives for system root, control set, and drive mappings |
