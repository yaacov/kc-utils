# drivers -- driver copy and registry registration

Handles copying VirtIO driver files into a mounted Windows guest filesystem and updating the Windows registry so the drivers are discovered at boot. Boot-critical storage drivers (viostor, vioscsi) are additionally placed in `system32\drivers` so they load before PnP enumeration. The DevicePath registry value is extended to include the VirtIO directory so Windows can locate INF files for plug-and-play installation.

Driver registration in the SYSTEM hive is delegated to pluggable `DriverRegistrar` implementations, selected by the version handler's `DriverRegistrarName()`. Each registrar knows how to write the correct registry entries (service keys, CriticalDeviceDatabase, etc.) for its class of Windows versions. PCI device IDs for boot-critical storage drivers are defined as constants for use by registrars.

## File layout

| File | Purpose |
|------|---------|
| `copy.go` | Copies `.sys`, `.inf`, `.cat`, and `.msi` files from the host driver tree into the guest |
| `devicepath.go` | Appends the VirtIO directory to the SOFTWARE hive `DevicePath` value |
| `pciids.go` | Defines `PCIIDPair` and `StoragePCIIDs` constants for viostor/vioscsi PCI device matching |
| `registrar.go` | Declares the `DriverRegistrar` interface and its plugin registry |

## Key exports

| Symbol | Role |
|--------|------|
| `Copy` | Copies driver files into `Windows\Drivers\VirtIO` and boot-critical `.sys` files into `system32\drivers` |
| `Update` | Appends `%SystemRoot%\Drivers\VirtIO` to the SOFTWARE hive `DevicePath` if not already present |
| `SCSIClassGUID` | Windows SCSI adapter class GUID constant used in CriticalDeviceDatabase entries |
| `PCIIDPair` | Struct holding legacy and modern VirtIO PCI IDs for a storage driver |
| `StoragePCIIDs` | Map of boot-critical driver names to their `PCIIDPair` values |
| `DriverRegistrar` | Interface for writing driver service/device registry entries into the SYSTEM hive |
| `Registrars` | Plugin registry of `DriverRegistrar` implementations keyed by name |
