# Direct Backend (host-mount)

`--backend=direct` / `V2V_backend=direct`. Registered on **Linux only**.

The direct backend mounts guest filesystems directly in the host kernel
namespace via `mount(8)`, then runs guest commands with `chroot` into that
mount (for example `dracut` / `grub-mkconfig` during convert). All disk I/O
goes through the host kernel's filesystem and block layers with no
intermediary.

```text
Host kernel
  ├── kc-prepare        mount(8) → guest FS at /tmp/kc-guest/
  ├── kc-convert-*      chroot /tmp/kc-guest … (and file I/O under that tree)
  └── kc-finalize       fstrim, umount(8), fsck
```

Implementation: [`pkg/guest/plugins/direct/`](../../pkg/guest/plugins/direct/README.md).

## Privileged capabilities

The following operations require elevated Linux capabilities:

| Operation | Tool / Syscall | Binary | Capability |
|-----------|----------------|--------|------------|
| Mount guest filesystems | `mount(8)` | kc-prepare | `CAP_SYS_ADMIN` |
| Unmount guest filesystems | `umount(8)` | kc-finalize | `CAP_SYS_ADMIN` |
| Loop device setup | `losetup`, `partx` | kc-prepare | `CAP_SYS_ADMIN` |
| LVM volume activation | `pvscan`, `vgscan`, `vgchange`, `lvscan` | kc-prepare | `CAP_SYS_ADMIN` |
| LVM deactivation | `vgchange -an` | kc-finalize | `CAP_SYS_ADMIN` |
| LUKS decrypt / close | `cryptsetup open` / `close` | kc-prepare / kc-finalize | `CAP_SYS_ADMIN` |
| Clevis LUKS unlock | `clevis luks unlock` (see [clevis-nbde.md](clevis-nbde.md)) | kc-prepare | `CAP_SYS_ADMIN` |
| Filesystem check | see [filesystem-checks.md](../architecture/filesystem-checks.md) | kc-prepare, kc-finalize | `CAP_SYS_ADMIN` |
| Filesystem trim | `fstrim` | kc-finalize | `CAP_SYS_ADMIN` |
| Chroot into guest | `chroot` (grub-mkconfig, dynamic scripts) | kc-convert-linux, kc-finalize | `CAP_SYS_CHROOT`, or `unshare -r` fallback |

The converter binaries (`kc-convert-linux`, `kc-convert-windows`) do not perform
mount or device operations themselves — they access guest disks exclusively
through `pkg/guest/`. `kc-convert-linux` runs guest commands (dracut,
grub-mkconfig) via `guest.RunInGuest`, which the direct backend implements as
`chroot`. `kc-convert-windows` reads virtio-win drivers from the pre-extracted
host-side tree at `/usr/share/virtio-win/drivers/by-os/` and copies them into
the guest via the `Guest` handle; it does not read that driver tree from
guest-disk I/O.

## Trade-offs

**Advantages:**

- Near-native I/O performance (direct kernel mount, no RPC overhead)
- No VM boot latency
- No KVM or hardware-virtualization requirement
- Smaller container image (no QEMU, no kernel, no appliance)
- Simpler debugging (guest files are visible in the host mount namespace)

**Disadvantages:**

- Requires `CAP_SYS_ADMIN` or `privileged: true` in Kubernetes pods
- Guest filesystem drivers run in the host kernel (less isolation)
- LVM, LUKS, and loop device state is shared with the host

## Pod security

`--backend` / `V2V_backend` is required (no CLI default). Direct is registered
on Linux only. The conversion pod needs `privileged: true` or, at minimum,
`CAP_SYS_ADMIN` with access to block device nodes. In Forklift deployments, the
pod security context is set by the Forklift operator, not by kc-utils itself.

See also [guestfs.md](guestfs.md) and [qemu.md](qemu.md) for the appliance
backends that avoid host `CAP_SYS_ADMIN`, and the
[comparison table](README.md#comparison).
