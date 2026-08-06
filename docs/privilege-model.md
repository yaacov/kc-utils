# Privilege Model: Host-Mount vs. libguestfs Appliance

In host-mount mode, `kc-prepare` and `kc-finalize` require root (or
`CAP_SYS_ADMIN`) because they mount guest filesystems directly in the host
kernel namespace. Guestfs mode avoids that by mounting guest filesystems inside
a QEMU appliance (`guestfish`) and performing all guest file I/O over guestfs
RPC — no host mounts and no FUSE. This document explains both approaches and
the trade-offs.

## Why kc-prepare / kc-finalize Need Root

The following operations all require `CAP_SYS_ADMIN` or equivalent:

| Operation | Tool / Syscall | Binary |
|-----------|----------------|--------|
| Mount guest filesystems | `mount(8)` | kc-prepare |
| Unmount guest filesystems | `umount(8)` | kc-finalize |
| Loop device setup | `losetup`, `partx` | kc-prepare |
| LVM volume activation | `pvscan`, `vgscan`, `vgchange`, `lvscan` | kc-prepare |
| LVM deactivation | `vgchange -an` | kc-finalize |
| LUKS decrypt / close | `cryptsetup open` / `close` | kc-prepare / kc-finalize |
| Clevis LUKS unlock | `clevis luks unlock` | kc-prepare |
| Filesystem check | `e2fsck`, `xfs_repair`, `btrfs`, `ntfsfix` | kc-prepare, kc-finalize |
| Filesystem trim | `fstrim` | kc-finalize |
| Chroot into guest | `chroot` (grub-mkconfig, dynamic scripts) | kc-convert-linux, kc-finalize |

The converter binaries (`kc-convert-linux`, `kc-convert-windows`) do not perform
mount or device operations themselves — they access guest disks exclusively
through `pkg/guest/`. `kc-convert-linux` runs guest commands (dracut,
grub-mkconfig) via `guest.RunInGuest`, which the direct backend implements as
`chroot` and the guestfs backend implements as an in-appliance shell.
`kc-convert-windows` reads virtio-win drivers from the pre-extracted tree at
`/usr/share/virtio-win/drivers/by-os/` on the host filesystem (not guest-disk I/O).
filesystem (no loop device; works in unprivileged pods) — this is not guest-disk
I/O.

## Host-Mount Approach (default)

kc-utils mounts guest filesystems directly on the host via `mount(8)`.
All disk I/O goes through the host kernel's filesystem and block layers with no
intermediary.

```
Host kernel
  ├── kc-prepare        mount(8) → guest FS visible at /tmp/kc-guest/
  ├── kc-convert-*      read/write files under /tmp/kc-guest/
  └── kc-finalize       fstrim, umount(8), fsck
```

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

## libguestfs Appliance Approach

[libguestfs](https://libguestfs.org/) avoids host-kernel mounts entirely by
running all filesystem operations inside a lightweight QEMU virtual machine
called the "appliance."

### How It Works

1. **Appliance construction.** [supermin](https://libguestfs.org/supermin.1.html)
   builds a minimal Linux kernel + initramfs containing filesystem tools (mount,
   LVM, cryptsetup, fsck, etc.). This is cached and reused across invocations.

2. **QEMU launch.** libguestfs starts a QEMU process with the appliance kernel.
   Guest disks are attached to the VM as virtio-blk or virtio-scsi devices.

3. **guestfsd daemon.** Inside the appliance, a daemon (`guestfsd`) listens on
   a virtio-serial channel. It accepts RPC commands from the host-side library
   and executes them as the appliance VM's own root.

4. **Host-side API.** kc-utils drives the appliance with `guestfish` (CLI),
   preferring `virt-guestfish` when present (see [NTFS on RHEL/UBI](#ntfs-mounts-on-rhelcentosubi)
   below). Each command — mount, download/upload, fsck, fstrim, etc. — is sent
   to `guestfsd` over virtio-serial and run inside the VM.

```
kc-v2v / kc-prepare / kc-finalize (unprivileged)
  └── virt-guestfish (--listen / --remote)   # or guestfish when no symlink
        └── QEMU (appliance VM, LIBGUESTFS_BACKEND=direct)
              ├── Linux kernel (appliance's own)
              ├── guestfsd (root inside VM)
              │     mount, LVM, LUKS, fsck, fstrim ...
              └── virtio-serial ←→ guestfish RPC
```

Because mount, LVM, and LUKS operations happen inside the appliance VM, the
host process never calls `mount(8)` on guest disks. "Root" inside the appliance is just
pid 1 of a throwaway VM -- it grants no privileges on the host.

kc-utils always uses `LIBGUESTFS_BACKEND=direct`: libguestfs launches QEMU
itself. That needs `/dev/kvm` in the pod; it does not use libvirtd.

## Comparison

| | Host-mount (kc-utils) | libguestfs appliance |
|-|----------------------|---------------------|
| **Host privileges** | `CAP_SYS_ADMIN` / `privileged: true` | `/dev/kvm` access only |
| **Performance** | Near-native (direct kernel I/O) | Appliance boot (~2-5 s) + virtio-serial RPC latency |
| **KVM required** | No | Yes (or very slow TCG software emulation) |
| **Container image size** | Smaller (no QEMU, no kernel) | Larger (QEMU + appliance kernel + initramfs) |
| **Isolation** | Weak (guest FS drivers in host kernel) | Strong (VM boundary) |
| **Complexity** | Go code + standard Linux tools | Supermin build, QEMU lifecycle, guestfsd RPC |
| **Debugging** | Easy (files visible under mount root) | Harder (must go through guestfs API or appliance shell) |

## Pod Security Implications

### Host-mount (default for CLI / unset `V2V_guestfs`)

The conversion pod needs `privileged: true` or, at minimum, `CAP_SYS_ADMIN`
with access to block device nodes. In Forklift deployments, the pod security
context is set by the Forklift operator, not by kc-utils itself.

### libguestfs appliance (`--guestfs` / `V2V_guestfs=true`)

Default in the `kc-v2v` container image (`ENV V2V_guestfs=true`). The
conversion pod needs `/dev/kvm` access (typically via the KubeVirt device
plugin `devices.kubevirt.io/kvm`). No `CAP_SYS_ADMIN`, privileged mode, or
`/dev/fuse` is required.

kc-utils guestfs mode uses one shared `guestfish --listen` appliance with a
strict ownership model:

1. **`kc-v2v` starts** the listener and continues only once the listen socket is ready.
2. **prepare / convert / finalize adopt** that session via `GUESTFISH_PID` /
   `KC_GUESTFISH_PID` and talk to disks only through `pkg/guest`. Stages never
   start or exit the shared listener (a dead shared PID is an error, not a
   cue to launch a second appliance).
3. **`kc-v2v` closes** the listener after finalize returns (success or failure
   cleanup), via `guestfish --remote exit`.

Standalone `kc-prepare` / `kc-finalize` (no PID env) may start a process-local
listener and exit it on `Release` / `Teardown`.

Discovery, probe, Guest FS RPC, trim, and fsck reuse the session via
`--remote` instead of launching QEMU for every call. Guest filesystems stay
mounted inside the appliance. Stage code reads and writes through `pkg/guest`
(`ReadFile`, `WriteFile`, `Exists`, …) over guestfish RPC. Tools that need a
real host path (for example hivex on Windows registry hives) use
`Guest.Checkout` / `Checkin` to download a single file to a temp path and
upload it back. `guestmount` (FUSE) is not used. `Guest.Sync()` is a no-op —
writes already hit the appliance-mounted filesystems.

### NTFS mounts on RHEL/CentOS/UBI

**Symptom.** With `libguestfs-winsupport` installed, mounting an NTFS volume
via guestfish still fails:

```text
libguestfs: error: mount: unsupported filesystem type
```

Root probe then finds no Windows hives and prepare exits with
`no root device found in guest`.

**Cause.** RHEL/CentOS (and UBI images that use those libguestfs builds) ship
a distro patch (RHBZ#1240276; also described in the
[libguestfs FAQ](https://libguestfs.org/guestfs-faq.1.html)) that allowlists
NTFS `mount` / `mount_ro` / `mount_options` / `mount_vfs` only when the
libguestfs handle’s program name starts with `virt-`. That was intended for
`virt-v2v` / `virt-p2v`. Plain `guestfish` sets the program from `argv[0]` to
`guestfish`, so NTFS mounts are rejected even though winsupport bits are in
the appliance.

**Distro difference.** Fedora and Debian libguestfs do **not** apply this
restriction; plain `guestfish` can mount NTFS there.

**Workaround in kc-utils.** The `kc-v2v` image installs a symlink:

```text
/usr/bin/virt-guestfish → guestfish
```

`pkg/guest/guestfs` prefers that binary via `guestfishBinary()` for
`--listen`, `--remote`, and scripts, so `argv[0]` is `virt-guestfish` and the
allowlist accepts NTFS mounts. No `set-program` call is required when every
process is started as `virt-guestfish`.

**Check.** `make test-kc-v2v-image` asserts the symlink (and, with `/dev/kvm`
and `REQUIRE_GUESTFS=1`, a real NTFS create/mount).
