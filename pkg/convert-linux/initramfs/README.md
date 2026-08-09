# initramfs -- virtio module injection into initramfs

Rebuilds the guest's initramfs image to include virtio drivers required for early boot under KubeVirt. Without these drivers baked into the initramfs, the converted VM cannot mount its root filesystem or bring up its network interface during the initial boot stage.

`InjectVirtioModules` backs up the existing initramfs and attempts to rebuild it using `dracut` with a fixed list of virtio and display drivers (`virtio`, `virtio_ring`, `virtio_blk`, `virtio_scsi`, `virtio_net`, `virtio_pci`, `xts`, `bochs-drm`, `bochs`). Dracut silently skips modules absent from the kernel tree, so no pre-filtering is needed. After dracut completes, a verification step compares the new initramfs size against the backup to detect silent failures. If dracut is unavailable or fails, the function falls back to Debian's `update-initramfs` or `mkinitramfs`, first ensuring the virtio module names are listed in `/etc/initramfs-tools/modules`. The initrd path is either taken from the kernel metadata or inferred by checking common naming conventions (`initramfs-<ver>.img`, `initrd.img-<ver>`, `initrd-<ver>`).

## Key exports

| Symbol | Role |
|--------|------|
| `InjectVirtioModules` | Rebuilds the initramfs for a selected kernel with virtio drivers, falling back through dracut, update-initramfs, and mkinitramfs |
