# uefi plugins

`UEFIEditor` interface — read and modify UEFI boot configuration.
`uefi.ConvertAllESPs` runs every registered editor against each ESP.

UEFI boot entries often reference hypervisor-specific bootloaders or paths that
do not exist after conversion to KVM. Each editor inspects the EFI System
Partition (ESP) and rewrites or supplements boot entries so the guest boots
correctly under the new hypervisor. The shared `ConvertAllESPs` helper mounts
each ESP device in turn and runs every registered editor against it.

| Key | Package | Editor |
|-----|---------|--------|
| `bcd` | bcdeditor/ | Windows BCD store editor via hivexregedit |
| `grub-fallback` | ../../convert-linux/uefi/plugins/grubfallback/ | Linux shim/grub fallback bootloader on ESP |

## bcd (bcdeditor)

**What it does:** Patches the Windows Boot Configuration Data (BCD) store on
the ESP so that Windows finds a valid boot entry after conversion.

**How it works:** The editor locates the BCD hive file on the ESP
(`EFI/Microsoft/Boot/BCD`), opens it using the `hivex` registry editor, and
rewrites boot object entries to point to the correct OS loader paths. Changes
are applied via batched `hivexregedit --merge` commands through the shared
`pkg/common/registry/hivex` writer.

## grub-fallback (grubfallback)

**What it does:** Ensures UEFI firmware can locate a GRUB or shim bootloader
on the ESP after the source hypervisor's boot files have been removed.

**How it works:** The editor scans the ESP for existing shim and GRUB EFI
binaries (e.g. `shimx64.efi`, `grubx64.efi`). When the standard fallback path
(`EFI/BOOT/BOOTX64.EFI`) is missing or points to a hypervisor-specific loader,
the editor copies the shim or GRUB binary into the fallback location. This
guarantees that the UEFI firmware's default boot path finds a working
bootloader regardless of which hypervisor-specific entries existed before
conversion.
