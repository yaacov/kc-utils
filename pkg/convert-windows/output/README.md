# output -- GuestCaps builder and permission fixup

Builds the `GuestCaps` structure that describes the converted guest's device capabilities and fixes file permissions on firstboot scripts. The capabilities inform downstream tooling (e.g. KubeVirt VM creation) which bus types, virtio features, and machine type to use.

`Build` inspects the list of copied driver names to determine bus types: if viostor or vioscsi was copied, `BlockBus` is set to "virtio" (otherwise "ide"); if netkvm was copied, `NetBus` is set to "virtio" (otherwise "e1000"). Additional boolean capabilities (RNG, balloon, socket, PVPanic, Virtio 1.0) are derived similarly. The machine type is selected based on architecture ("q35" for x86_64, "virt" for aarch64). `FixPermissions` walks the Guestfs firstboot directory tree on the mounted guest filesystem and sets standard Unix permissions (755 for directories, 644 for files).

## File layout

| File | Purpose |
|------|---------|
| `guestcaps.go` | Populates `GuestCaps` fields from copied driver names and guest architecture |
| `postconvert.go` | Walks the Guestfs firstboot directory and fixes file/directory permissions |

## Key exports

| Symbol | Role |
|--------|------|
| `Build` | Fills `GuestCaps` block/net bus, virtio feature flags, and machine type from copied driver names |
| `FixPermissions` | Sets 755/644 permissions on all directories/files under the Guestfs firstboot tree |
