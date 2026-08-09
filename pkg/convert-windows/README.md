# kc-convert-windows blocks

All pipeline blocks for [`cmd/kc-convert-windows`](../../cmd/kc-convert-windows/main.go). Pluggable
blocks document implementers in `<block>/plugins/README.md`. Each block has its
own README with detailed exports and mechanism.

| # | Block | Package | Type | Description |
|---|-------|---------|------|-------------|
| 1 | Version | [`version/`](version/) | strict | Classify Windows version, select behavior handlers |
| 2 | Driver Source | [`driversource/`](driversource/) | pluggable | Find virtio-win drivers on conversion host |
| 3 | Inspect | [`inspect/`](inspect/) | strict | Antivirus detection, RTC mode |
| 4 | Hypervisor | [`hypervisor/`](hypervisor/) | pluggable | Remove hypervisor software, disable services |
| 5 | Drivers | [`drivers/`](drivers/) | strict + pluggable | Copy virtio drivers, register in registry |
| 6 | Crash Control | [`crashcontrol/`](crashcontrol/) | strict | Disable auto-reboot on BSOD |
| 7 | Firstboot | [`firstboot/`](firstboot/) | strict | PowerShell/batch firstboot scripts |
| 8 | NTFS Fix | [`ntfsfix/`](ntfsfix/) | strict | Patch NTFS boot sector for pre-Vista Windows |
| 9 | UEFI | `pkg/common/uefi/` | pluggable | Update UEFI boot entries |
| 10 | Output | [`output/`](output/) | strict | Build GuestCaps, fix firstboot dir permissions |
| — | Static IP | [`staticip/`](staticip/) | helper | PowerShell/registry static IP script generation |

Orchestrator: [`pkg/cmd/convert-windows/`](../cmd/convert-windows/).
Docs: [`docs/apps/kc-convert-windows.md`](../../docs/apps/kc-convert-windows.md).
