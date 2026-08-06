# kc-convert-linux blocks

All pipeline blocks for [`cmd/kc-convert-linux`](../../cmd/kc-convert-linux/main.go). Pluggable
blocks document implementers in `<block>/plugins/README.md`.

| # | Block | Package | Type | Description |
|---|-------|---------|------|-------------|
| 1 | Distro | [`distro/`](distro/) | pluggable | Classify OS, package format, package manager |
| 2 | Bootloader | [`bootloader/`](bootloader/) | pluggable | Detect boot config format (grub2, bls) |
| 3 | Remap | [`remap/`](remap/) | pluggable | Rewrite block device names in fstab/crypttab/bootloader |
| 4 | Kernel | [`kernel/`](kernel/) | pluggable scan + strict select | Scan kernels, select best virtio candidate |
| 5 | Boot Config | [`bootconfig/`](bootconfig/) | strict | Serial console and virtio video kernel args |
| 6 | UEFI | [`uefi/`](uefi/) | pluggable | Update UEFI boot entries on ESP |
| 7 | Hypervisor | [`hypervisor/`](hypervisor/) | pluggable | Remove source hypervisor tools |

Hypervisor cleanup plugins (VMware, Xen, VirtualBox, Parallels, Citrix, …):
[`hypervisor/plugins/README.md`](hypervisor/plugins/README.md).

| 8 | Guest Agent | [`guestagent/`](guestagent/) | pluggable | qemu-guest-agent, static IP, local packages |
| 9 | Guest Cleanup | [`guestcleanup/`](guestcleanup/) | strict | Remove blkid/LVM caches, update modprobe aliases |
| 10 | Initramfs | [`initramfs/`](initramfs/) | strict | Inject virtio modules via pure-Go CPIO |
| 11 | NIC Naming | [`nicnaming/`](nicnaming/) | pluggable | Preserve NIC names and static IP configuration |
| 12 | GuestCaps | [`guestcaps/`](guestcaps/) | strict | Derive block/net bus, virtio flags, machine type |

## Kernel sub-packages

[`kernel/`](kernel/) scans installed kernels and picks the best virtio candidate:

| File / plugin | Role |
|---|---|
| `kernel.go` | Orchestrate scan + select |
| `select.go` | Score kernels for virtio module support |
| [`plugins/rpm/`](kernel/plugins/rpm/) | RPM-based kernel enumeration |
| [`plugins/deb/`](kernel/plugins/deb/) | dpkg-based kernel enumeration |

## Bootconfig sub-packages

[`bootconfig/`](bootconfig/) patches kernel command line; grub2 handlers regenerate
config via `grub2-mkconfig` / `grub-mkconfig` when defaults change:

| File | Role |
|---|---|
| [`console.go`](bootconfig/console.go) | Serial console (`console=ttyS0`) |
| [`display.go`](bootconfig/display.go) | Virtio video / disable legacy VGA |

## Guest cleanup sub-packages

[`guestcleanup/`](guestcleanup/) removes stale guest state after conversion:

| File | Role |
|---|---|
| `guestcleanup.go` | Entry point (`Clean` + `Configure`) |
| `modalias.go` | Update modprobe aliases for virtio |

## Initramfs injection

[`initramfs/`](initramfs/) uses pure-Go CPIO manipulation:

1. Decompress existing initramfs (gzip/xz/zstd via [`pkg/common/compression/`](../common/compression/))
2. Parse CPIO archive
3. Append virtio `.ko` modules from guest filesystem
4. Update module dependency metadata
5. Recompress and write back

Orchestrator: [`pkg/cmd/convert-linux/`](../cmd/convert-linux/).
Docs: [`docs/kc-convert-linux.md`](../../docs/kc-convert-linux.md).

Import path prefix: `github.com/yaacov/kc-utils/pkg/convert-linux/…`
