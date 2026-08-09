# firmware plugins

`FirmwareDetector` interface — confidence-based winner selects BIOS vs UEFI.

Firmware detection determines whether the guest uses BIOS or UEFI boot, which
affects how the finalize pipeline resolves the target firmware type for
KubeVirt and how the UEFI conversion blocks decide whether to run. The
detector runs twice in the prepare pipeline: once before mounting (using raw
partition metadata) and once after mounting (when mount points are known),
with the post-mount result taking precedence since it has richer information.

| Key | Package | Description |
|-----|---------|-------------|
| `gpt-esp` | gptesp/ | Heuristic ESP detection (`vfat` + mount point / size / path); returns UEFI when an ESP-like partition is found |

## gpt-esp

**What it does:** Determines whether the guest uses UEFI firmware by searching
for an EFI System Partition (ESP) across all attached disks.

**How it works:** Iterates every partition on every disk, applying a heuristic
to decide whether it looks like an ESP. A partition is classified as an ESP
candidate when it has `FSType == "vfat"` and meets any of these criteria:

- Its mount point is `/boot/efi` or `/efi`.
- It is the first partition on the disk and is 1 GiB or smaller (typical ESP
  sizing).
- Its device path contains the substring `efi` (case-insensitive).

When a matching partition is found, the detector returns a `FirmwareInfo` with
type `uefi` and the partition's device path in the `ESPDevices` list. If no
ESP candidate is found across any disk, it returns type `bios`. The default
fallback (when no detector is registered or all detectors fail) is also BIOS.
