# mount plugins

`MountPlanner` interface — plan guest filesystem mounts from fstab or Windows layout.

Mount planning builds the ordered list of filesystems that need to be mounted
so the conversion pipeline can access the full guest filesystem tree. The
planner runs in two phases: `Plan` returns the root mount, and `Expand` (called
after the root is mounted) discovers additional mounts from the guest's own
configuration. Mount specs are sorted by path depth so parent directories are
mounted before their children.

| Key | Package | Description |
|-----|---------|-------------|
| `linux` | fstab/ | Parse `/etc/fstab` and mount Linux guest filesystems |
| `windows` | windows/ | Mount Windows system hive and boot partitions |

## fstab (linux)

**What it does:** Plans filesystem mounts for Linux guests by parsing the
guest's `/etc/fstab`.

**How it works:** The `Plan` method returns a single `MountSpec` for the root
device (`/`), detecting the filesystem type via `blkid` if needed. After the
root is mounted, `Expand` reads `etc/fstab` from the mounted guest, parses it
using the `configedit/fstab` package, and resolves each device reference
(`UUID=`, `LABEL=`, `/dev/disk/by-*`) to an actual device path via a
`resolve.Catalog`. Entries for swap, pseudo-filesystems (proc, sysfs, tmpfs,
devtmpfs, devpts), and paths under `/proc`, `/sys`, `/dev` are skipped. The
resulting mount specs are sorted by path length so nested mounts (`/boot`
before `/boot/efi`) are applied in the correct order.

## windows

**What it does:** Plans filesystem mounts for Windows guests, mounting the
system partition and optionally the EFI System Partition.

**How it works:** The `Plan` method returns the root NTFS mount at `/` plus,
when UEFI firmware is detected, the ESP mounted at `/boot/efi`. The ESP is
located by checking `Firmware.ESPDevices` from the prepare metadata first,
then falling back to scanning the same disk for a `vfat` partition. The
`Expand` method is a no-op since Windows has no fstab equivalent — all
necessary mounts are determined from the partition layout during the initial
plan phase.
