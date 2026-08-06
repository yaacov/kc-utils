# kc-prepare blocks

All pipeline blocks for [`cmd/kc-prepare`](../../cmd/kc-prepare/main.go). Pluggable
blocks register implementers under `plugins/` (see each block's `plugins/README.md`).

| # | Block | Package | Type | Description |
|---|-------|---------|------|-------------|
| 1 | Validate | [`validate/`](validate/) | strict | Validate disk list and create mount root |
| 2 | Guest | [`guest/`](guest/) | strict | Open disks, scan partitions, activate LVM |
| 3 | Decrypt | inline (`pkg/guest/`) | inline | LUKS decryption (after open/LVM; `all` tries every device) |
| 4 | Pre-Fsck | inline (`pkg/guest/`) | inline | Pre-conversion fsck |
| 5 | Firmware | [`firmware/`](firmware/) | pluggable | BIOS vs UEFI detection (also refreshed after mount) |
| 6 | Root | [`root/`](root/) | strict + pluggable | Root discovery and selection (default `first`) |
| 7 | Mount | [`mount/`](mount/) | strict + pluggable | Mount planning and execution |
| 8 | Inspect | [`inspect/`](inspect/) | strict | OS inspection, boot device, free space |
| 9 | Converter | [`converter/`](converter/) | pluggable | Choose linux/windows converter |

## Guest sub-packages

[`guest/`](guest/) replaces libguestfs with host `mount(8)` and a `Guest` I/O API.

| Sub-package | Role |
|-------------|------|
| [`guest/luks/`](guest/luks/) | LUKS1/2 decrypt via `anatol/luks.go`, `containers/luksy`, or `cryptsetup` |
| [`guest/overlay/`](guest/overlay/) | qcow2 overlay on block devices (kc-v2v `V2V_overlayEnabled`) |
| [`guest/resolve/`](guest/resolve/) | blkid UUID/label → device path catalog for fstab remapping |

## Other block internals

| Block | Sub-package / file | Role |
|---|---|---|
| Inspect | [`inspect/inspect.go`](inspect/inspect.go) | OS type, distro, version detection; Windows registry-backed product/version metadata |
| Inspect | [`inspect/freespace.go`](inspect/freespace.go) | Mounted guest free-space checks and recorded stats for `/`, `/boot`, and `/boot/efi` when present |
| Root | [`root/discover.go`](root/discover.go) | Scan disks for bootable OS roots |
| Root | [`root/plugins/`](root/plugins/) | Root selection policy (`first`, `single`, `device`) |
| Mount | [`mount/plan.go`](mount/plan.go) | Mount ordering (nested paths first) |
| Mount | [`mount/plugins/fstab/`](mount/plugins/fstab/) | Plan mounts from `/etc/fstab` |

Orchestrator: [`pkg/cmd/prepare/`](../cmd/prepare/).
Docs: [`docs/kc-prepare.md`](../../docs/kc-prepare.md).
