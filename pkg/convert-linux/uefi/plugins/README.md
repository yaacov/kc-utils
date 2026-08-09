# uefi plugins

`UEFIEditor` interface — update UEFI boot entries on ESP partitions.

On UEFI-booted Linux guests the EFI System Partition may contain
hypervisor-specific boot entries or lack the standard fallback bootloader path.
This plugin ensures the ESP has a working GRUB or shim binary at the UEFI
fallback path so the guest boots after conversion without depending on
NVRAM boot variables (which are not preserved across hypervisor migration).

| Key | Package | Description |
|-----|---------|-------------|
| `grub-fallback` | grubfallback/ | Copy shim/grub fallback files so UEFI firmware finds a bootloader |

## grub-fallback

**What it does:** Copies a shim or GRUB EFI binary into the UEFI fallback
bootloader path (`EFI/BOOT/BOOTX64.EFI` or the aarch64 equivalent) on the ESP.

**How it works:** Scans the ESP for existing `shimx64.efi`, `shimaa64.efi`,
`grubx64.efi`, or `grubaa64.efi` binaries under distro-specific directories
(e.g. `EFI/redhat/`, `EFI/ubuntu/`, `EFI/BOOT/`). If the standard fallback
path is missing or contains a hypervisor-specific loader, the plugin copies the
best available shim or GRUB binary into place. This ensures the UEFI firmware
boots the correct chain (shim → GRUB → kernel) without relying on NVRAM
entries that may have been lost during migration.
