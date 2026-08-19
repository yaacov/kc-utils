# qemu Backend: Our Own Minimal Appliance

The `qemu` backend is a third guest-disk access mode alongside
[host-mount (`direct`) and libguestfs (`guestfs`)](privilege-model.md). It boots
a **purpose-built, minimal appliance we build ourselves** directly with
`qemu-system-*` — no libvirt, no libguestfs, no supermin — and talks to a tiny
in-guest agent over a unix socket.

The defining principle: **the appliance exposes only primitive operations; all
conversion logic stays host-side.** Partition discovery, LVM activation, LUKS /
Clevis unlock, mount planning, filesystem checks, and chrooted guest commands are
all *composed on the host* out of a small primitive vocabulary, then executed
inside the appliance via the agent.

## Why a third backend

| | `direct` | `guestfs` | `qemu` |
|-|----------|-----------|--------|
| Host privileges | `CAP_SYS_ADMIN` | `/dev/kvm` | qemu (KVM/HVF/TCG) |
| Dependency | host kernel + tools | libguestfs/supermin stack | just `qemu-system-*` + our image |
| Host OS | Linux | Linux | **Linux or macOS** |
| We control the appliance | n/a | no (upstream) | **yes** |

The host side only launches `qemu-system-*` and speaks a unix socket — every
Linux tool runs *inside* the appliance — so `qemu` is a first-class backend on
macOS (Apple Silicon via HVF), not just Linux.

## Topology

```text
kc-v2v / kc-prepare / kc-convert-* / kc-finalize   (host, unprivileged)
  └── pkg/backend/plugins/qemu  (all conversion logic composed here)
        └── qemu-system-<arch>  (-kernel vmlinuz -initrd initramfs.img)
              ├── guest disks   → virtio-blk → /dev/vd{a,b,…}
              └── virtio-serial ←──unix socket──→ kc-guest-agent (/init, PID 1)
                    primitives only: exec, file I/O, raw dev I/O, stat/statfs
```

Disks attach in order, so the host maps disk *i* deterministically to
`/dev/vd{a+i}` — no device-listing round trip. The agent port is named
`org.kc-utils.agent`, bridged to a host unix socket that QEMU owns (`-chardev
socket,...,server=on`). The named node `/dev/virtio-ports/org.kc-utils.agent` is
created by udev, which the minimal appliance does not run, so the agent resolves
the port from `/sys/class/virtio-ports/*/name` and opens the raw `/dev/vportNpM`.

## Wire protocol

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

## Host composes, appliance executes

The host builds the *logic*; the appliance only runs the *tools*. Examples:

| Host-side logic (this package) | Appliance primitive(s) it uses |
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

## Acceleration & architecture

`accelFor(GOOS, hasAccel)` picks **KVM** on Linux, **HVF** on macOS, else **TCG**.
With acceleration the machine uses `-cpu host`; under TCG a concrete model
(`cortex-a72` on arm64, `max` on x86_64). An arm64 appliance boots near-native on
an arm64 Mac; an x86_64 appliance runs under TCG there.

`RunCommand` chroots into the guest and runs the guest's *own* (guest-arch)
binaries. Converting an x86_64 guest therefore needs an x86_64 appliance — on
Apple Silicon that means TCG for the convert stage.

## VM/session lifecycle

- **Boot** (`session.go`): launch qemu, then poll `Ping` on a fresh connection
  until the agent answers or boot times out (longer under TCG). Console output is
  captured in a bounded buffer for diagnostics; an early process exit fails boot
  fast.
- **Shared across stages**: `kc-v2v` boots one appliance with all disks attached
  (`StartSharedListener(disks)`) and exports `KC_QEMU_SOCK` / `KC_QEMU_PID`. Each
  stage subprocess **adopts** it (`adoptVMSession`) rather than booting its own,
  so mounts established in one stage persist into the next — exactly like the
  guestfs shared listener. This is required for a full multi-stage pipeline
  (mounts live inside the VM).
- **Standalone**: a single-stage run with no shared socket boots its own VM and
  re-establishes mounts from the recorded disk infos (`remountFromDiskInfos`).
- **Crash recovery**: an owned session can `restart()` once (adopted sessions
  cannot — the parent owns the process).
- **Close**: an owned session powers off / SIGTERMs the VM and removes the
  socket; an adopted session only drops its client connection.

## Building the appliance

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
backend (darwin + linux); build its appliance separately with `make
build-appliance` and point `KC_APPLIANCE_DIR` at the output (or install it under
`/usr/lib/kc-utils/appliance/<arch>/`).

## Selection

`--backend qemu` (or `V2V_backend=qemu`). Availability requires a
`qemu-system-*` binary **and** the appliance image for the selected appliance
architecture — the host `GOARCH` by default, or `KC_APPLIANCE_ARCH` when set.
Hardware acceleration is preferred but not required (TCG fallback), so the
backend's `Requirements` ask for `QEMU` but not `Accel`.
