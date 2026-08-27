# initramfs -- virtio module injection into initramfs

Rebuilds the guest's initramfs image to include virtio drivers required for early boot under KubeVirt. Without these drivers baked into the initramfs, the converted VM cannot mount its root filesystem or bring up its network interface during the initial boot stage.

`InjectVirtioModules` backs up the existing initramfs and rebuilds it with
`dracut --add-drivers` for virtio boot modules (`virtio`, `virtio_ring`,
`virtio_blk`, `virtio_scsi`, `virtio_net`, `virtio_pci`, `xts`). That flag is
one `dracut-install -m` list: a missing module (for example `bochs_drm` on
RHEL 9 server kernels) fails the whole set, so optional display drivers are
not included. Dracut can still exit 0 after printing `dracut: FAILED`; that
output is treated as failure. A size check against the backup catches silent
no-ops. If dracut is unavailable or fails, the function falls back to Debian's
`update-initramfs` or `mkinitramfs`, first ensuring the virtio module names are
listed in `/etc/initramfs-tools/modules`. The initrd path is either taken from
the kernel metadata or inferred by checking common naming conventions
(`initramfs-<ver>.img`, `initrd.img-<ver>`, `initrd-<ver>`).

## Key exports

| Symbol | Role |
|--------|------|
| `InjectVirtioModules` | Rebuilds the initramfs for a selected kernel with virtio drivers, falling back through dracut, update-initramfs, and mkinitramfs |
