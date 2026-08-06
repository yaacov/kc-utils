# kc-convert-windows blocks

All pipeline blocks for [`cmd/kc-convert-windows`](../../cmd/kc-convert-windows/main.go). Pluggable
blocks document implementers in `<block>/plugins/README.md`.

| # | Block | Package | Type | Description |
|---|-------|---------|------|-------------|
| 1 | Driver Source | [`driversource/`](driversource/) | pluggable | Find virtio-win drivers on conversion host |
| 2 | Inspect | [`inspect/`](inspect/) | strict | Antivirus detection, RTC mode |
| 3 | Hypervisor | [`hypervisor/`](hypervisor/) | pluggable | Remove hypervisor software, disable services |

Hypervisor plugins (VMware, EC2, Nutanix, AWS PV, …):
[`hypervisor/plugins/README.md`](hypervisor/plugins/README.md).

| 4 | Drivers | [`drivers/`](drivers/) | strict + pluggable | Copy virtio drivers, register in registry |
| 5 | Crash Control | [`crashcontrol/`](crashcontrol/) | strict | Disable auto-reboot on BSOD |
| 6 | Firstboot | [`firstboot/`](firstboot/) | strict | PowerShell firstboot scripts |
| 7 | NTFS Fix | [`ntfsfix/`](ntfsfix/) | strict | Patch NTFS boot sector for pre-Vista Windows |
| 8 | UEFI | `pkg/common/uefi/` | pluggable | Update UEFI boot entries |
| 9 | Output | [`output/`](output/) | strict | Build GuestCaps, fix firstboot dir permissions |

## Inspect sub-packages

[`inspect/`](inspect/) collects Windows-specific pre-conversion metadata:

| File | Role |
|---|---|
| [`antivirus.go`](inspect/antivirus.go) | Detect AV products that may block driver install |
| [`rtcmode.go`](inspect/rtcmode.go) | Read RTC UTC/local mode from registry |

## Drivers sub-packages

[`drivers/`](drivers/) copies virtio-win files into the guest and registers them:

| File / plugin | Role |
|---|---|
| [`copy.go`](drivers/copy.go) | Copy driver files from host virtio-win tree |
| [`devicepath.go`](drivers/devicepath.go) | Resolve guest device paths for driver binding |
| [`registrar.go`](drivers/registrar.go) | Write driver entries to registry hives |
| [`plugins/driverdb/`](drivers/plugins/driverdb/) | Match drivers from virtio-win driver database |
| [`plugins/criticaldb/`](drivers/plugins/criticaldb/) | Install critical/boot-start drivers first |

## Driver source

VirtIO drivers are read from the **conversion host** pre-extracted tree at
`/usr/share/virtio-win/drivers/by-os/`. See
[`driversource/plugins/README.md`](driversource/plugins/README.md) for paths and
driver tree population.

Registry access uses [`pkg/common/registry/`](../common/registry/) — pure Go read,
`hivexregedit` for writes.

Supporting helper (not a pipeline block): [`staticip/`](staticip/) — PowerShell firstboot
for preserving static IP across NIC change.

Orchestrator: [`internal/convert-windows/`](../../internal/convert-windows/).
Docs: [`docs/kc-convert-windows.md`](../../docs/kc-convert-windows.md).
