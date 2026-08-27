# Guest Disk Backends

kc-utils accesses guest disks through three pluggable backends registered in
`pkg/backend/plugins/`. Pipeline stages (`kc-prepare`, `kc-convert-*`,
`kc-finalize`) never call `mount(8)` or `guestfish` directly — they use
`pkg/guest`, which the selected backend implements.

Selection is explicit: `--backend direct|guestfs|qemu` on stage CLIs, or
`V2V_backend` in the `kc-v2v` container (default `guestfs` in the release
image; CLI default is `direct`).

## Requirements

Each backend declares runtime prerequisites via `backend.Requirements` in
`pkg/backend/plugin.go`. `pkg/backend/runtime.go` probes the host at startup.

| Backend | `--backend` | Declared requirements | Runtime probes |
|---------|-------------|----------------------|----------------|
| **direct** | `direct` (CLI default) | Linux, root or `CAP_SYS_ADMIN` | `runtime.GOOS == linux`, euid 0 or `CapEff` bit 21 |
| **guestfs** | `guestfs` | Linux, `/dev/kvm`, `guestfish` | Linux, `/dev/kvm` readable, `virt-guestfish` or `guestfish` in `PATH` |
| **qemu** | `qemu` | `qemu-system-*` binary | `qemu-system-<arch>` or `qemu-kvm` in `PATH`, plus appliance image for the selected arch (`KC_APPLIANCE_ARCH`, default host `GOARCH`) under `KC_APPLIANCE_DIR` |

Hardware acceleration (KVM on Linux, HVF on macOS) is preferred for appliance
backends but not required — `guestfs` falls back to TCG when `/dev/kvm` is
missing (slow), and `qemu` always falls back to TCG when no accelerator is
present (`Accel` is not in `Requirements` for `qemu`).

See also [pkg/backend/plugins/README.md](../../pkg/backend/plugins/README.md).

## Shared pipeline model

Converter binaries (`kc-convert-linux`, `kc-convert-windows`) do not perform
mount or device operations themselves — they access guest disks exclusively
through `pkg/guest/`. `kc-convert-linux` runs guest commands (`dracut`,
`grub-mkconfig`) via `guest.RunInGuest`; `kc-convert-windows` reads virtio-win
drivers from the pre-extracted host-side tree at
`/usr/share/virtio-win/drivers/by-os/` and copies them into the guest via the
`Guest` handle.

**Read a guest file** (`guest.ReadFile("/etc/fstab")`):

```bash
# direct: live path under the mount root
cat /tmp/kc-guest/etc/fstab

# guestfs: RPC to the shared listener (socket /tmp/.guestfish-$UID/socket-$PID)
# GUESTFISH_PID is set by kc-v2v; stages use guestfish --remote=$PID
guestfish --remote=$GUESTFISH_PID -- download /etc/fstab /tmp/kc-fstab.$$
cat /tmp/kc-fstab.$$

# qemu: framed JSON over KC_QEMU_SOCK (host composes ReadFile primitive)
```

**Run a command in the guest** (`guest.RunCommand(..., []string{"dracut", "--force"})`):

```bash
# direct: chroot into the mounted tree (needs CAP_SYS_CHROOT or unshare -r chroot)
chroot /tmp/kc-guest /usr/bin/dracut --force

# guestfs: guestfish "sh" inside the already-mounted appliance
guestfish --remote=$GUESTFISH_PID -- sh 'dracut --force 2>&1'

# qemu: bind /proc /sys /dev, Exec chroot <root> … inside the appliance
```

### Shared listener sessions

`guestfs` and `qemu` keep a VM-resident session alive across pipeline stages
when orchestrated by `kc-v2v`:

1. **`kc-v2v` starts** the listener and continues only once the socket is ready.
2. **prepare / convert / finalize adopt** that session via `GUESTFISH_PID` /
   `KC_GUESTFISH_PID` or `KC_QEMU_SOCK` / `KC_QEMU_PID`. Stages never start or
   exit a shared listener owned by the parent (a dead shared PID is an error).
3. **`kc-v2v` closes** the listener after finalize returns (success or failure).

Standalone `kc-prepare` / `kc-finalize` (no PID/socket env) may start a
process-local listener and exit it on `Release` / `Teardown`.

In `guestfs` mode there is no populated host tree at `/tmp/kc-guest`; that path
is only a key for `pkg/guest` helpers. All disk I/O goes over the listen
socket (`/tmp/.guestfish-<uid>/socket-<pid>`).

## direct (host-mount)

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

### Operations that need elevated privileges

`kc-prepare` and `kc-finalize` call host tools that require `CAP_SYS_ADMIN` or
equivalent:

| Operation | Tool / Syscall | Binary |
|-----------|----------------|--------|
| Mount guest filesystems | `mount(8)` | kc-prepare |
| Unmount guest filesystems | `umount(8)` | kc-finalize |
| Loop device setup | `losetup`, `partx` | kc-prepare |
| LVM volume activation | `pvscan`, `vgscan`, `vgchange`, `lvscan` | kc-prepare |
| LVM deactivation | `vgchange -an` | kc-finalize |
| LUKS decrypt / close | `cryptsetup open` / `close` | kc-prepare / kc-finalize |
| Clevis LUKS unlock | `clevis luks unlock` | kc-prepare |
| Filesystem check | see [filesystem-checks.md](filesystem-checks.md) | kc-prepare, kc-finalize |
| Filesystem trim | `fstrim` | kc-finalize |
| Chroot into guest | `chroot` (grub-mkconfig, dynamic scripts) | kc-convert-linux, kc-finalize |

## guestfs (libguestfs appliance)

[libguestfs](https://libguestfs.org/) avoids host-kernel mounts entirely by
running all filesystem operations inside a lightweight QEMU virtual machine
called the "appliance."

### How it works

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

Discovery, probe, Guest FS RPC, trim, and fsck reuse the session via
`--remote` instead of launching QEMU for every call. Guest filesystems stay
mounted inside the appliance during convert; finalize unmounts before fsck.
NTFS FSCheck uses guestfish `ntfsfix` on the block device (requires
`ntfs3g=yes` in the appliance — same as NTFS mount); see
[filesystem-checks.md](filesystem-checks.md). Stage code reads and writes through
`pkg/guest` over guestfish RPC. Tools that need a real host path (for example
hivex on Windows registry hives) use `Guest.Checkout` / `Checkin` to download a
single file to a temp path and upload it back. `guestmount` (FUSE) is not used.
`Guest.Sync()` is a no-op — writes already hit the appliance-mounted filesystems.

## qemu

The `qemu` backend boots a minimal appliance with `qemu-system-*` and talks to
`kc-guest-agent` over a virtio-serial unix socket. The appliance exposes only
primitive operations; partition discovery, LVM activation, LUKS / Clevis unlock,
mount planning, filesystem checks, and chrooted guest commands are composed on
the host and executed inside the appliance via the agent.

```text
kc-v2v / kc-prepare / kc-convert-* / kc-finalize   (host)
  └── pkg/backend/plugins/qemu  (conversion logic composed here)
        └── qemu-system-<arch>  (-kernel vmlinuz -initrd initramfs.img)
              ├── guest disks   → virtio-blk → /dev/vd{a,b,…}
              └── virtio-serial
                    ├── org.kc-utils.agent ←──unix socket──→ kc-guest-agent protocol
                    │     primitives only: exec, file I/O, raw dev I/O, stat/statfs
                    └── org.kc-utils.debug ←──debug.sock──→ interactive bash (PTY)
```

The in-appliance binary is [`kc-guest-agent`](../apps/kc-guest-agent.md).
Disks attach in order, so the host maps disk *i* deterministically to
`/dev/vd{a+i}` — no device-listing round trip. The agent port is named
`org.kc-utils.agent`, bridged to a host unix socket that QEMU owns (`-chardev
socket,...,server=on`). A second port, `org.kc-utils.debug`, is bridged to
`debug.sock` in the same directory for an interactive shell (see
[Interactive debug shell](#interactive-debug-shell)). The named node
`/dev/virtio-ports/<name>` is created by udev, which the minimal appliance does
not run, so the agent resolves ports from `/sys/class/virtio-ports/*/name` and
opens the raw `/dev/vportNpM`.

### Wire protocol

`pkg/qemuagent/proto` — length-prefixed JSON frames: a 4-byte big-endian length
followed by one JSON object. `[]byte` fields are base64 (readability over wire
efficiency; payloads are config files, boot sectors, individual driver files).
One flat `Request`, one flat `Response`. Primitive ops:

```text
Ping  Exec  ReadFile  WriteFile  Stat  ReadDir  Mkdir  Remove
Rename  Symlink  Readlink  Chmod  PRead  PWrite  StatFS
```

`Exec` runs an executable in the appliance and returns stdout/stderr/exit code.
Everything else is straightforward file / raw-device / metadata I/O. The agent
(`pkg/qemuagent/server`) is portable Go (`os` / `os/exec`); it never imports
`pkg/backend`, and its handlers are testable on any OS over `net.Pipe`.

### Host composes, appliance executes

The host builds the *logic*; the appliance only runs the *tools*. Examples:

| Host-side logic (qemu package) | Appliance primitive(s) it uses |
|-------------------------------|--------------------------------|
| partition discovery (`discover.go`) | `Exec lsblk -J`; parsed host-side |
| LVM activation | `Exec pvscan --cache`, `vgchange -ay`, `lvs` |
| mount planning + rebasing (`mount.go`) | `Exec mkdir`, `Exec mount` |
| LUKS unlock (`crypt.go`) | `WriteFile` keyfile, `Exec cryptsetup open` |
| Clevis/NBDE unlock | `Exec clevis luks unlock` (+ user-net at launch) |
| fs-check (`fscheck.go`) | `Exec e2fsck` / `xfs_repair` / `btrfs check` / `ntfsfix` |
| guest file I/O (`fs.go`) | `ReadFile` / `WriteFile` / `ReadDir` / … |
| run guest command (`run.go`) | bind `/proc /sys /dev`, `Exec chroot <root> …` |
| OS probe (`probe.go`) | RO-mount, `Download` OS markers to a host temp dir |

Guest paths are rebased under the appliance mount root `/mnt/guest`
(`guestToAppliance`, `applianceMountPath`); raw device paths (`/dev/vd*`) are
**not** rebased.

### Acceleration and architecture

`accelFor(GOOS, hasAccel)` picks **KVM** on Linux, **HVF** on macOS, else **TCG**.
With acceleration the machine uses `-cpu host`; under TCG a concrete model
(`cortex-a72` on arm64, `max` on x86_64). An arm64 appliance boots near-native on
an arm64 Mac; an x86_64 appliance runs under TCG there.

`RunCommand` chroots into the guest and runs the guest's own binaries. Guest
kernel, arch, and OS still come from the mounted filesystem (inspect,
`DetectArch`, explicit `dracut` kernel version) — not from the appliance
`uname`. When the guest ELF ISA is not the appliance's, the agent has
registered **binfmt_misc** with qemu-user-static (`F` flag); chroot mounts
(`/proc`, `/sys`, `/dev`) are the same as same-ISA convert. `--no-hostonly`
keeps dracut from treating appliance `/sys` as the target machine.

The appliance defaults to host `GOARCH` (`KC_APPLIANCE_ARCH` unset), so an
Apple Silicon Mac can convert an x86_64 disk under HVF. Set
`KC_APPLIANCE_ARCH` to the guest arch to boot a same-ISA TCG appliance if
binfmt convert fails. First-cut interpreters are **x86_64/i386 ↔ aarch64**
only (ppc64le/s390x stay same-ISA).

guestfs and direct do not get this binfmt; foreign-ISA `RunInGuest` there still
needs a same-ISA environment (or host binfmt for direct).

### VM/session lifecycle

- **Boot** (`session.go`): launch qemu, then poll `Ping` on a fresh connection
  until the agent answers or boot times out (longer under TCG). Console output is
  captured in a bounded buffer for diagnostics; an early process exit fails boot
  fast.
- **Shared across stages**: `kc-v2v` boots one appliance with all disks attached
  (`StartSharedListener(disks)`) and exports `KC_QEMU_SOCK` / `KC_QEMU_PID` /
  `KC_QEMU_DEBUG_SOCK`. Each stage subprocess **adopts** it (`adoptVMSession`)
  rather than booting its own, so mounts established in one stage persist into
  the next.
- **Standalone**: a single-stage run with no shared socket boots its own VM and
  re-establishes mounts from the recorded disk infos (`remountFromDiskInfos`).
- **Crash recovery**: an owned session can `restart()` once (adopted sessions
  cannot — the parent owns the process).
- **Close**: an owned session powers off / SIGTERMs the VM and removes the
  socket; an adopted session only drops its client connection.

### Interactive debug shell

Local Mac/Linux how-to (fetch disks, hold the appliance across stages, boot the
converted guest): [../debug/README.md](../debug/README.md). Agent CLI:
[kc-guest-agent.md](../apps/kc-guest-agent.md).

A second virtio-serial port (`org.kc-utils.debug`) is always attached. PID 1
starts `/bin/bash -i` on a PTY only after a host client connects to
`debug.sock`, and waits for the next connect after disconnect. This is
independent of the framed agent protocol, so you can inspect the appliance
**while prepare (or convert/finalize) is running**.

The host socket sits next to the agent socket: `/tmp/kc-qemu-*/debug.sock`. Boot
logs include `debug_socket`; a parent process also exports `KC_QEMU_DEBUG_SOCK`.

1. Start a qemu-backed run as usual (`kc-prepare --backend qemu …` or `kc-v2v`
   with `V2V_backend=qemu`) and wait until logs show `qemu appliance ready`.
2. Find the socket: `"$KC_QEMU_DEBUG_SOCK"` if set, otherwise
   `ls /tmp/kc-qemu-*/debug.sock` and pick one (several runs can leave
   stale siblings).
3. Attach from a second terminal (raw TTY so the guest PTY owns line discipline):

```sh
# If unset: ls /tmp/kc-qemu-*/debug.sock and pick one
sock=$KC_QEMU_DEBUG_SOCK
if [ -z "$sock" ]; then
  sock=$(ls /tmp/kc-qemu-*/debug.sock 2>/dev/null | awk 'NR==1')
fi
socat UNIX-CONNECT:"$sock" STDIO,raw,echo=0,escape=0x1d
```

You are in the **appliance** (initramfs), not the converted guest. Once prepare
has mounted filesystems they appear under `/mnt/guest`. Guest `exit` / Ctrl-D
only ends that bash; QEMU keeps `debug.sock` open so the agent starts a new
shell and `socat` stays connected. Leave with **Ctrl-]** (`escape=0x1d`).
Kernel boot messages stay on `-serial stdio` (captured for diagnostics) and
are not this channel.

Changing the in-guest agent requires rebuilding the initramfs
(`make build-appliance`); host-side qemu launch changes alone are not enough.

### Building the appliance

[`build/kc-appliance`](../../build/kc-appliance) builds the kernel + initramfs
via a Containerfile (Fedora minimal + the block/fs/LVM/LUKS/Clevis toolbox + the
agent as `/init`), extracted to `bin/appliance/<arch>/{vmlinuz,initramfs.img}`.
The backend locates them under `KC_APPLIANCE_DIR`
(default `/usr/lib/kc-utils/appliance`).

```sh
make build-appliance                 # arm64 + amd64
build/kc-appliance/build.sh arm64    # one arch
```

The appliance is **not** bundled in the `kc-v2v` container image — that image
converts with the `guestfs` backend. The `qemu` backend is an opt-in local
backend; build its appliance separately with `make build-appliance` and point
`KC_APPLIANCE_DIR` at the output (or install it under
`/usr/lib/kc-utils/appliance/<arch>/`).

## Comparison

| | direct | guestfs | qemu |
|-|--------|---------|------|
| **Host privileges** | `CAP_SYS_ADMIN` / `privileged: true` | `/dev/kvm` access only | `qemu-system-*` + appliance image |
| **Performance** | Near-native (direct kernel I/O) | Appliance boot (~2–5 s) + virtio-serial RPC latency | Appliance boot + framed JSON RPC |
| **KVM / accel** | Not required | KVM preferred (TCG fallback) | KVM/HVF preferred (TCG fallback) |
| **Container image size** | Smaller (no QEMU, no kernel) | Larger (QEMU + appliance kernel + initramfs) | Appliance built separately |
| **Isolation** | Weak (guest FS drivers in host kernel) | Strong (VM boundary) | Strong (VM boundary) |
| **Windows LDM (dynamic disk)** | Error — use `qemu` | Error — use `qemu` | `ldmtool` in appliance |
| **BitLocker** | Error — use `qemu` | Error — use `qemu` | `cryptsetup --type bitlk` in appliance |
| **Storage Spaces** | Error | Error | Error |
| **Debugging** | Easy (files visible under mount root) | Harder (guestfish API or appliance shell) | Agent socket + debug PTY channel |

## Pod security (Forklift / MTV)

### direct (`V2V_backend=direct`)

The conversion pod needs `privileged: true` or, at minimum, `CAP_SYS_ADMIN`
with access to block device nodes. In Forklift deployments, the pod security
context is set by the Forklift operator, not by kc-utils itself.

### guestfs (`--backend guestfs` / `V2V_backend=guestfs`)

Default in the `kc-v2v` container image (`ENV V2V_backend=guestfs`). The
conversion pod needs `/dev/kvm` access (typically via the KubeVirt device
plugin `devices.kubevirt.io/kvm`). No `CAP_SYS_ADMIN`, privileged mode, or
`/dev/fuse` is required.

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

The `qemu` backend unlocks Clevis inside its appliance with the same Tang
reachability requirements (`Exec clevis luks unlock` + user networking at launch).

### Windows LDM and BitLocker (`--backend qemu`)

Windows **dynamic disks** (LDM) and **BitLocker**-encrypted volumes are supported
only on the `qemu` backend. The appliance runs `ldmtool create all` after virtio
disks attach, discovers `/dev/mapper/ldm_*` mapper devices, and probes them for
`Windows/System32/config/SYSTEM` like LVM logical volumes. `direct` and `guestfs`
fail fast at setup with `windows volume <kind> on <device> requires backend "qemu"`.

BitLocker passphrases are supplied in `PrepareInput.bitlocker.key_files` (same
`all` / per-device pattern as `luks`). kc-v2v scans `/etc/bitlocker` by default
(`V2V_BITLOCKER_DIR`). Unlock runs inside the appliance with
`cryptsetup open --type bitlk`; TPM-only volumes are not supported.

**Storage Spaces** is unsupported on every backend.

Rebuild the appliance after pulling LDM support: `make build-appliance` (adds
`libldm` / `ldmtool` to the initramfs).

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
NTFS there. `pkg/backend/plugins/guestfs` prefers `virt-guestfish` via `guestfishBinary()`
when present (for `--listen`, `--remote`, and scripts), so `argv[0]` satisfies
the RHEL allowlist. No `set-program` call is required when every process is
started as `virt-guestfish`.

**Check.** `make test-kc-v2v-image` asserts guestfish + NTFS support (and, with
`/dev/kvm` and `REQUIRE_GUESTFS=1`, a real NTFS create/mount).
