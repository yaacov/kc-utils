# kc-appliance — minimal utility guest for the `qemu` backend

This directory builds the **appliance** used by the `qemu` guest-disk backend
(`pkg/backend/plugins/qemu`): a purpose-built, minimal Fedora image that boots
directly under `qemu-system-*` and serves a small set of primitive operations to
the host over a virtio-serial port.

We own this appliance end to end. It carries only what the host needs to inspect
and mutate guest disks — the kernel, the virtio drivers, and the
block/filesystem/LVM/LUKS/Clevis/LDM toolbox — plus the in-guest agent
(`kc-guest-agent`) running as `/init`.

## Artifacts

The build produces, per architecture:

```
bin/appliance/<arch>/vmlinuz         Linux kernel
bin/appliance/<arch>/initramfs.img   gzip'd cpio root, agent as /init
```

`<arch>` uses Go naming (`arm64`, `amd64`) — the same value the backend derives
from `runtime.GOARCH` (or `KC_APPLIANCE_ARCH`). The backend looks for these under
`KC_APPLIANCE_DIR` (default `/usr/lib/kc-utils/appliance`).

## Building

```sh
# Both arches:
make build-appliance

# One arch:
build/kc-appliance/build.sh arm64
ARCHES=amd64 build/kc-appliance/build.sh
```

The build is a container build (`podman` or `docker`); no host kernel tooling is
required. `build.sh` builds the Containerfile's assembly stage and copies
`vmlinuz` + `initramfs.img` out with `create`+`cp` (works with a remote daemon
such as the macOS podman machine, where `--output type=local` is unsupported).

### Slimming

The Containerfile aggressively trims the image to what a disk-conversion utility
actually needs — the initramfs is what the kernel loads into RAM at boot, so
size matters. Removed with rationale (see the cleanup `RUN` step): the kernel's
ARM board device trees (`dtb/`, unused on QEMU's `virt` machine), `System.map`,
the dracut-built `/boot` initramfs, the RPM database, systemd (the agent is
`/init`), dracut, the udev hardware DB, console keymaps, docs/locales/licenses,
and kernel driver categories a virtio VM never instantiates (gpu, sound, usb,
hid, input, hwmon, …). Filesystem, block, md/LVM, crypto, and virtio modules are
kept. This takes the arm64 initramfs from ~184M to ~84M with no loss of function
(verified by `TestE2EApplianceRoundTrip`).

### Cross-architecture

Building an `amd64` appliance on an arm64 host (or vice-versa) needs binfmt/
qemu-user emulation registered:

- **macOS (podman machine):** already registered inside the VM — cross builds
  work out of the box.
- **Linux:** `podman run --rm --privileged tonistiigi/binfmt --install all`.

## How it boots

There is no separate root filesystem: the kernel unpacks `initramfs.img` into a
tmpfs and runs `/init` (the agent) as PID 1. On start the agent:

1. mounts `/proc`, `/sys`, `/dev` (devtmpfs), `/dev/pts` (devpts, for the debug
   PTY), `/run`, `/tmp`;
2. `modprobe`s the virtio drivers (`virtio_console` for the serial ports,
   `virtio_blk` for the disks, …);
3. starts an interactive debug channel on virtio-serial port
   `org.kc-utils.debug` (`/bin/bash -i` on a PTY, respawned on exit);
4. resolves the agent virtio-serial port and serves the protocol in
   `pkg/qemuagent/proto`, reopening the port after each stage disconnects. The
   named node `/dev/virtio-ports/<name>` is a udev artifact, so with no udevd
   the agent falls back to matching `/sys/class/virtio-ports/*/name` and opens
   the raw `/dev/vportNpM` device.

To attach a shell to a running appliance (for example while `kc-prepare
--backend qemu` is in progress), connect to `debug.sock` next to `agent.sock`.
See [Interactive debug shell](../../docs/architecture/backends.md#interactive-debug-shell)
and the local how-to in [docs/debug/README.md](../../docs/debug/README.md).

The host side (launch args, session lifecycle, disk→`/dev/vd*` mapping) lives in
`pkg/backend/plugins/qemu`. See `docs/architecture/backends.md` for the
full protocol and the host/guest logic split.

## What's inside (and why)

| Package(s)              | Provides                                   |
|-------------------------|--------------------------------------------|
| `kernel-core`, `kmod`   | vmlinuz, virtio modules, `modprobe`        |
| `util-linux`            | `lsblk`, `blkid`, `mount`, `fstrim`, `chroot` |
| `lvm2`                  | `pvscan`, `vgchange`, `lvs`                 |
| `cryptsetup`            | LUKS open/close, BitLocker (`--type bitlk`) |
| `libldm`                | `ldmtool` — Windows LDM (dynamic disk) assembly |
| `clevis`, `clevis-luks` | NBDE/Tang unlock (needs `-netdev` at launch) |
| `e2fsprogs`, `xfsprogs`, `btrfs-progs` | `e2fsck`, `xfs_repair`, `btrfs check` |
| `ntfs-3g`, `ntfsprogs`  | `ntfsfix`, NTFS mount                       |
| `dosfstools`            | vfat / ESP tooling                         |
| `glibc`, `bash`, `coreutils` | runtime for chrooted guest commands   |

Keep this list in sync with the host backend's expectations (`discover.go`,
`ldm.go`, `fscheck.go`, `crypt.go`, `run.go`).
