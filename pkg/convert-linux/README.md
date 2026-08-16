# kc-convert-linux blocks

All pipeline blocks for [`cmd/kc-convert-linux`](../../cmd/kc-convert-linux/main.go). Pluggable
blocks document implementers in `<block>/plugins/README.md`. Each block has its
own README with detailed exports and mechanism.

Stage-local helpers (not pipeline blocks): [`systemd/`](systemd/) — shared systemd unit mask/disable utilities used by hypervisor plugins and network handlers.

**Type:** `strict` = single built-in implementation · `pluggable` = implementation chosen from a `plugins/` registry ([plugin model](../../community/architecture.md#plugin-system)) · `strict + pluggable` = registry plus built-in wiring/fallback · `inline` = handled directly by the stage orchestrator.

| # | Block | Package | Type | Description |
|---|-------|---------|------|-------------|
| 1 | Distro | [`distro/`](distro/) | pluggable | Classify OS family |
| 2–3 | Package format / manager | [`distro/`](distro/) | strict | RPM/deb format and dnf/apt/zypper |
| 4 | Bootloader | [`bootloader/`](bootloader/) | pluggable | Detect boot config format (grub2, bls) |
| 5 | Kernel scan | [`kernel/`](kernel/) | pluggable | Scan installed kernels |
| 6 | Remap | [`remap/`](remap/) | pluggable | Rewrite block device names in fstab/crypttab/bootloader |
| 7 | UEFI | [`uefi/`](uefi/) + `pkg/common/uefi/` | pluggable | Update UEFI boot entries on ESP |
| 8 | Kernel select | [`kernel/`](kernel/) | strict | Select best virtio-capable kernel |
| 9–10 | Boot config | [`bootconfig/`](bootconfig/) | strict | Serial console and virtio video kernel args |
| 11 | Hypervisor | [`hypervisor/`](hypervisor/) | pluggable | Remove source hypervisor tools |
| 11b, 15 | Network | [`network/`](network/) | pluggable (`Select`) | Exclusive handler: networkd offline config or default firstboot path |
| 12 | Guest agent | [`guestagent/`](guestagent/) | pluggable | qemu-guest-agent, static IP firstboot, local packages |
| 13 | Guest cleanup | [`guestcleanup/`](guestcleanup/) | strict | Remove blkid/LVM caches, update modprobe aliases |
| 14 | Initramfs | [`initramfs/`](initramfs/) | strict | Rebuild initramfs with virtio drivers |
| 16 | SELinux | [`selinux/`](selinux/) | strict | Offline SELinux relabel via setfiles |
| 17 | GuestCaps | [`guestcaps/`](guestcaps/) | strict | Derive block/net bus, virtio flags, machine type |

Orchestrator: [`pkg/cmd/convert-linux/`](../cmd/convert-linux/).
Docs: [`docs/apps/kc-convert-linux.md`](../../docs/apps/kc-convert-linux.md).

Import path prefix: `github.com/yaacov/kc-utils/pkg/convert-linux/…`
