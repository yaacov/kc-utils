# remap -- block device name remapping interface

Defines the `DeviceRemapper` interface for rewriting block device names in guest configuration files during conversion. When a VM moves between hypervisors, device names change (e.g. `/dev/sda` to `/dev/vda`), and references in fstab, GRUB configs, and other files must be updated accordingly.

Concrete `DeviceRemapper` implementations register into a global plugin registry. Each provides `Name` to identify itself, `Detect` to check whether the guest has configuration files it can handle, and `Remap` to perform the actual device name rewriting.

## Key exports

| Symbol | Role |
|--------|------|
| `DeviceRemapper` | Interface for rewriting block device names in guest configs (Name, Detect, Remap) |
| `Remappers` | Global plugin registry of `DeviceRemapper` implementations |
