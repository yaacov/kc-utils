# drivers plugins

`DriverRegistrar` interface — register copied VirtIO drivers in the Windows registry.

After the driver files have been copied into the guest's `Windows\Drivers\VirtIO\`
directory (by the strict copy block), each registrar plugin writes the
necessary Windows registry entries so the boot manager loads the storage
drivers early enough to find the system disk. Without correct registry
entries, Windows would BSOD on first boot because it cannot access the
virtio-backed disk. Two registrar plugins exist because the Windows driver
registration mechanism changed between legacy versions (CriticalDeviceDatabase)
and modern versions (DriverDatabase). The `version.VersionHandler` determines
which registrar name to use for a given Windows version.

| Key | Package | Description |
|-----|---------|-------------|
| `criticaldb` | criticaldb/ | CriticalDeviceDatabase keys (`PCI#VEN_…&DEV_…&REV_…`) for legacy Windows |
| `driverdb` | driverdb/ | DriverDatabase (`DriverInfFiles` / `DeviceIds` / packages) plus Services keys (Windows 8+) |

Strict block also:

- Copies drivers to `Windows\Drivers\VirtIO\`
- Copies boot `.sys` files to `system32\drivers\` (ImagePath points there)
- Stages `qemu-ga*.msi` for firstboot install when present in `DriverFiles` (see [`CollectGuestAgentMSI`](../../version/guestagent.go))
- Updates DevicePath to include `\Drivers\VirtIO`

## criticaldb

**What it does:** Registers VirtIO storage drivers in the Windows
`CriticalDeviceDatabase` registry area, which is the pre-Windows 10 mechanism
for ensuring boot-critical drivers are loaded at kernel initialization.

**How it works:** Creates a service key under
`{CurrentControlSet}\Services\{driverName}` with `Type=1` (kernel driver),
`Start=0` (boot-start), `ErrorControl=1`, and the driver image path. For
known storage drivers (`viostor`, `vioscsi`), also creates
`CriticalDeviceDatabase` entries under
`{CurrentControlSet}\Control\CriticalDeviceDatabase\PCI#{vendorDeviceId}` with
the SCSI class GUID and service name, for both legacy and modern PCI device
IDs (VEN_1AF4&DEV_1001/1042 for viostor, VEN_1AF4&DEV_1004/1048 for vioscsi).

## driverdb

**What it does:** Registers VirtIO drivers in the Windows `DriverDatabase`
registry area, which is the Windows 10+ mechanism for PnP driver management.

**How it works:** Creates a service key with `Start=0` (boot-start) for
boot-critical drivers (`viostor`, `vioscsi`) or `Start=3` (demand-start) for
others. For storage drivers, creates three sets of registry entries:

- `DriverDatabase\DriverInfFiles\{driver}.inf` — marks the INF as active with
  its configuration name.
- `DriverDatabase\DeviceIds\PCI\{vendorDeviceId}` — binds PCI hardware IDs to
  the driver INF.
- `DriverDatabase\DriverPackages\{infLabel}` — writes a version blob,
  configuration binding, and descriptors.

If a service key already exists (e.g. from a prior install), it is left
unchanged unless the driver is boot-critical, in which case `Start` is forced
to `0` to ensure early loading.
