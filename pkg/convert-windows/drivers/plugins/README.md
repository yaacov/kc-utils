# drivers plugins

`DriverRegistrar` interface — register copied VirtIO drivers in the Windows registry.

| Key | Package | Description |
|-----|---------|-------------|
| `criticaldb` | criticaldb/ | CriticalDeviceDatabase keys (`PCI#VEN_…&DEV_…&REV_…`) for legacy Windows |
| `driverdb` | driverdb/ | DriverDatabase (`DriverInfFiles` / `DeviceIds` / packages) plus Services keys (Windows 8+) |

Strict block also:

- Copies drivers to `Windows\Drivers\VirtIO\`
- Copies boot `.sys` files to `system32\drivers\` (ImagePath points there)
- Stages `qemu-ga*.msi` for firstboot install
- Updates DevicePath to include `\Drivers\VirtIO`
