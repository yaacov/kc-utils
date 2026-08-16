# Privilege Model: Host-Mount vs. guestfish (libguestfs appliance)

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
| Clevis LUKS unlock (direct) | `clevis luks unlock` | kc-prepare |
| Clevis LUKS unlock (guestfs) | guestfish `clevis-luks-unlock` + appliance network (appliance-root inside the VM; not host `CAP_SYS_ADMIN`) | kc-prepare |
| Filesystem check | see [filesystem-checks.md](filesystem-checks.md) | kc-prepare, kc-finalize |
| Filesystem trim | `fstrim` | kc-finalize |
| Chroot into guest | `chroot` (grub-mkconfig, dynamic scripts) | kc-convert-linux, kc-finalize |

The converter binaries (`kc-convert-linux`, `kc-convert-windows`) do not perform
mount or device operations themselves — they access guest disks exclusively
through `pkg/guest/`. `kc-convert-linux` runs guest commands (dracut,
grub-mkconfig) via `guest.RunInGuest`, which the direct backend implements as
`chroot` and the guestfs backend implements as an in-appliance shell.
`kc-convert-windows` reads virtio-win drivers from the pre-extracted host-side tree at
`/usr/share/virtio-win/drivers/by-os/` and copies them into the guest via the `Guest` handle; it does not read that driver tree from guest-disk I/O.

## Host-Mount Approach (default)

kc-utils mounts guest filesystems directly on the host via `mount(8)`, then
runs guest commands with `chroot` into that mount (for example `dracut` /
`grub-mkconfig` during convert). All disk I/O goes through the host kernel's
filesystem and block layers with no intermediary.

```text
Host kernel
  ├── kc-prepare        mount(8) → guest FS at /tmp/kc-guest/
  ├── kc-convert-*      chroot /tmp/kc-guest … (and file I/O under that tree)
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

```text
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

### Same Go call, two backends

Pipeline code always goes through `pkg/guest` (`ReadFile`, `RunCommand`, …).
The backend chooses host paths vs. the guestfish listen socket.

**Read a guest file** (`guest.ReadFile("/etc/fstab")`):

```bash
# Host-mount (direct): live path under the mount root
cat /tmp/kc-guest/etc/fstab

# Guestfs: RPC to the shared listener (socket /tmp/.guestfish-$UID/socket-$PID)
# GUESTFISH_PID is set by kc-v2v; stages use guestfish --remote=$PID
guestfish --remote=$GUESTFISH_PID -- download /etc/fstab /tmp/kc-fstab.$$
cat /tmp/kc-fstab.$$
```

**Run a command in the guest** (`guest.RunCommand(..., []string{"dracut", "--force"})`):

```bash
# Host-mount (direct): chroot into the mounted tree (needs CAP_SYS_CHROOT
# or unshare -r chroot fallback)
chroot /tmp/kc-guest /usr/bin/dracut --force

# Guestfs: guestfish "sh" inside the already-mounted appliance
# (virtual /proc, /sys, /dev are mounted for the call, then unmounted)
guestfish --remote=$GUESTFISH_PID -- sh 'dracut --force 2>&1'
```

In guestfs mode there is no populated host tree at `/tmp/kc-guest`; that path
is only a key for `pkg/guest` helpers. All disk I/O goes over the listen
socket (`/tmp/.guestfish-<uid>/socket-<pid>`).

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

### Host-mount (`--backend=direct` / `V2V_backend=direct`)

CLI default when `V2V_backend` / `--backend` is unset is `direct`. The
conversion pod needs `privileged: true` or, at minimum, `CAP_SYS_ADMIN`
with access to block device nodes. In Forklift deployments, the pod security
context is set by the Forklift operator, not by kc-utils itself.

### libguestfs appliance (`--backend=guestfs` / `V2V_backend=guestfs`)

Default in the `kc-v2v` container image (`ENV V2V_backend=guestfs`). The
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
mounted inside the appliance during convert; finalize unmounts before fsck.
NTFS FSCheck uses guestfish `ntfsfix` on the block device (requires
`ntfs3g=yes` in the appliance — same as NTFS mount); see
[filesystem-checks.md](filesystem-checks.md) for the full matrix. Stage code
reads and writes through `pkg/guest`
(`ReadFile`, `WriteFile`, `Exists`, …) over guestfish RPC. Tools that need a
real host path (for example hivex on Windows registry hives) use
`Guest.Checkout` / `Checkin` to download a single file to a temp path and
upload it back. `guestmount` (FUSE) is not used. `Guest.Sync()` is a no-op —
writes already hit the appliance-mounted filesystems.

### Clevis / NBDE (Forklift `V2V_NBDE_CLEVIS`)

Forklift sets `V2V_NBDE_CLEVIS=true` on the conversion pod when Plan
`nbdeClevis` or Conversion `diskEncryption.type=Clevis` is configured. LUKS
passphrase secrets are mounted at `/etc/luks` (no Clevis env). Clevis takes
precedence over keyfiles when both are present.

In guestfs mode, unlock uses guestfish `clevis-luks-unlock` inside the
appliance (not host `clevis`). That requires:

1. Appliance networking enabled before `run` (`set-network true`), gated on
   Clevis via internal `KC_GUESTFS_NETWORK=1`.
2. Tang servers from the Clevis pin tree reachable from the conversion pod
   network (QEMU user networking).
3. The `clevisluks` libguestfs feature (clevis packages available to supermin).

After unlock, prepare rescans LVM and probes `/dev/mapper/*` devices as root
candidates. Keyfile unlock uses `cryptsetup-open` with `--keys-from-stdin`.

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

**Workaround in kc-utils.** On RHEL/UBI images, install a symlink:

```text
/usr/bin/virt-guestfish → guestfish
```

The upstream Fedora `kc-v2v` image does not need this; plain `guestfish` mounts
NTFS there. `pkg/guest/guestfs` prefers `virt-guestfish` via `guestfishBinary()`
when present (for `--listen`, `--remote`, and scripts), so `argv[0]` satisfies
the RHEL allowlist. No `set-program` call is required when every process is
started as `virt-guestfish`.

**Check.** `make test-kc-v2v-image` asserts guestfish + NTFS support (and, with
`/dev/kvm` and `REQUIRE_GUESTFS=1`, a real NTFS create/mount).
