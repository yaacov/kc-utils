# remap plugins

`DeviceRemapper` interface — rewrite block device names for KVM/virtio.

Source hypervisors use device names that do not exist under KVM (e.g.
`/dev/hda` for IDE on old Xen, `/dev/xvda` for Xen PV, `/dev/sda` for
VMware SCSI). After conversion the guest's disks appear as virtio devices
(`/dev/vda`). The remap block rewrites all persistent references to old
device names so the guest can boot and mount filesystems correctly.

| Key | Package | Description |
|-----|---------|-------------|
| `standard` | standard/ | Remap `sd`/`hd`/`xvd` prefixes in `/etc/fstab`, `/etc/crypttab`, grub defaults, and BLS entries; crypttab `/dev/sd*` prefers `UUID=` via blkid when available |

## standard

**What it does:** Rewrites block device name prefixes from hypervisor-specific
naming (`sd`, `hd`, `xvd`) to virtio naming (`vd`) across all configuration
files that contain persistent device references.

**How it works:** `Detect` checks whether the guest filesystem contains any
references to non-virtio device prefixes in the target files. `Remap` then
processes each file:

- **`/etc/fstab`** — rewrites `/dev/sda1` → `/dev/vda1` (and similar) using
  the `configedit/fstab` package. UUID- and LABEL-based entries are left
  untouched since they are device-name-independent.
- **`/etc/crypttab`** — rewrites device paths; when a `/dev/sd*` path is found,
  prefers replacing it with `UUID=` (via `blkid` lookup) for robustness.
- **GRUB defaults** (`/etc/default/grub`) — updates `GRUB_CMDLINE_LINUX` root
  device references via the `configedit/grub` package.
- **BLS entries** (`/boot/loader/entries/*.conf`) — updates `options` lines
  containing device paths via the `configedit/bls` package.
