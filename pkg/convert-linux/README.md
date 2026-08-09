# kc-convert-linux blocks

All pipeline blocks for [`cmd/kc-convert-linux`](../../cmd/kc-convert-linux/main.go). Pluggable
blocks document implementers in `<block>/plugins/README.md`. Each block has its
own README with detailed exports and mechanism.

| # | Block | Package | Type | Description |
|---|-------|---------|------|-------------|
| 1 | Distro | [`distro/`](distro/) | pluggable | Classify OS, package format, package manager |
| 2 | Bootloader | [`bootloader/`](bootloader/) | pluggable | Detect boot config format (grub2, bls) |
| 3 | Remap | [`remap/`](remap/) | pluggable | Rewrite block device names in fstab/crypttab/bootloader |
| 4 | Kernel | [`kernel/`](kernel/) | pluggable scan + strict select | Scan kernels, select best virtio candidate |
| 5 | Boot Config | [`bootconfig/`](bootconfig/) | strict | Serial console and virtio video kernel args |
| 6 | UEFI | [`uefi/`](uefi/) | pluggable | Update UEFI boot entries on ESP |
| 7 | Hypervisor | [`hypervisor/`](hypervisor/) | pluggable | Remove source hypervisor tools |
| 8 | Guest Agent | [`guestagent/`](guestagent/) | pluggable | qemu-guest-agent, static IP, local packages |
| 9 | Guest Cleanup | [`guestcleanup/`](guestcleanup/) | strict | Remove blkid/LVM caches, update modprobe aliases |
| 10 | Initramfs | [`initramfs/`](initramfs/) | strict | Inject virtio modules via pure-Go CPIO |
| 11 | NIC Naming | [`nicnaming/`](nicnaming/) | pluggable | Preserve NIC names and static IP configuration |
| 12 | SELinux | [`selinux/`](selinux/) | strict | Offline SELinux relabel via setfiles |
| 13 | GuestCaps | [`guestcaps/`](guestcaps/) | strict | Derive block/net bus, virtio flags, machine type |

Orchestrator: [`pkg/cmd/convert-linux/`](../cmd/convert-linux/).
Docs: [`docs/apps/kc-convert-linux.md`](../../docs/apps/kc-convert-linux.md).

Import path prefix: `github.com/yaacov/kc-utils/pkg/convert-linux/…`
