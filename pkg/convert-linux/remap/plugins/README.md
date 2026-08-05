# remap plugins

`DeviceRemapper` interface — rewrite block device names for KVM/virtio.

| Key | Package | Description |
|-----|---------|-------------|
| `standard` | standard/ | Remap `sd`/`hd`/`xvd` prefixes in `/etc/fstab`, `/etc/crypttab`, grub defaults, and BLS entries; crypttab `/dev/sd*` prefers `UUID=` via blkid when available |
