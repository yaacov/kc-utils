# Guestfs Backend (libguestfs appliance)

`--backend=guestfs` / `V2V_backend=guestfs`. Registered on **Linux only**.
Default in the `kc-v2v` container image (`ENV V2V_backend=guestfs`).

[libguestfs](https://libguestfs.org/) avoids host-kernel mounts entirely by
running all filesystem operations inside a lightweight QEMU virtual machine
called the "appliance." The conversion pod needs `/dev/kvm` access only — no
`CAP_SYS_ADMIN`, privileged mode, or `/dev/fuse`.

Implementation: [`pkg/guest/plugins/guestfs/`](../../pkg/guest/plugins/guestfs/README.md).

## How it works

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
host process never calls `mount(8)` on guest disks. "Root" inside the appliance
is just pid 1 of a throwaway VM — it grants no privileges on the host.

kc-utils always uses `LIBGUESTFS_BACKEND=direct`: libguestfs launches QEMU
itself. That needs `/dev/kvm` in the pod; it does not use libvirtd.

> This is the **libguestfs/supermin** appliance. The `qemu` backend uses a
> different appliance built by kc-utils — see [appliance.md](appliance.md).

## Same Go call, two backends

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

## Shared-session ownership

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
[filesystem-checks.md](../architecture/filesystem-checks.md) for the full matrix.
Stage code reads and writes through `pkg/guest` (`ReadFile`, `WriteFile`,
`Exists`, …) over guestfish RPC. Tools that need a real host path (for example
hivex on Windows registry hives) use `Guest.Checkout` / `Checkin` to download a
single file to a temp path and upload it back. `guestmount` (FUSE) is not used.
`Guest.Sync()` is a no-op — writes already hit the appliance-mounted
filesystems.

## Pod security

Default in the `kc-v2v` container image (`ENV V2V_backend=guestfs`). The
conversion pod needs `/dev/kvm` access (typically via the KubeVirt device
plugin `devices.kubevirt.io/kvm`). No `CAP_SYS_ADMIN`, privileged mode, or
`/dev/fuse` is required.

For Clevis/NBDE unlock inside the appliance (appliance networking,
`KC_GUESTFS_NETWORK`), see [clevis-nbde.md](clevis-nbde.md).

## NTFS mounts on RHEL/CentOS/UBI

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
libguestfs handle's program name starts with `virt-`. That was intended for
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
NTFS there. `pkg/guest/plugins/guestfs` prefers `virt-guestfish` via `guestfishBinary()`
when present (for `--listen`, `--remote`, and scripts), so `argv[0]` satisfies
the RHEL allowlist. No `set-program` call is required when every process is
started as `virt-guestfish`.

**Check.** `make test-kc-v2v-image` asserts guestfish + NTFS support (and, with
`/dev/kvm` and `REQUIRE_GUESTFS=1`, a real NTFS create/mount).
