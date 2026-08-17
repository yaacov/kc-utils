# The QEMU Appliance

The [`qemu` backend](qemu.md) boots a purpose-built appliance VM whose pid 1 is
[`kc-agent`](kc-agent.md). This page describes that appliance — what it is, how
it is built, and how the backend finds it at runtime.

> Not to be confused with the **libguestfs/supermin appliance** used by the
> [`guestfs` backend](guestfs.md). That one is built and cached by libguestfs;
> the appliance described here is built by kc-utils and shipped as two files.

## Artifacts

`make appliance` produces two files **per architecture** under
`build/appliance/out/<arch>/`:

| File | What it is |
|------|------------|
| `vmlinuz` | Fedora kernel |
| `initramfs.img` | xz-compressed RAM root: virtio + filesystem kernel modules, conversion tools, and `kc-agent` as pid 1 |

Virtio-win and qemu-guest-agent RPMs are **not** packed into the appliance —
they stay on the conversion host (`/usr/share/virtio-win`,
`/usr/share/kc-packages`; see [qemu.md](qemu.md#requirements)).

## Building

```bash
make appliance          # builds both amd64 and arm64
make appliance-amd64    # linux/amd64 only
make appliance-arm64    # linux/arm64 only
```

`make appliance-<arch>` invokes `appliance-arch` with the matching
`PLATFORM`, which runs a container build of
[`build/appliance/Containerfile`](../../build/appliance/Containerfile) and
writes the output to `$(APPLIANCE_OUT)/<arch>` (default
`build/appliance/out/<arch>`). A container runtime (podman/docker) is required.

The build has two stages:

1. **kc-agent** — `CGO_ENABLED=0 GOOS=linux GOARCH=<arch>` static build of
   [`cmd/kc-agent`](kc-agent.md).
2. **pack** — a Fedora rootfs installs the conversion toolchain (`lvm2`,
   `cryptsetup`, `clevis`/`clevis-luks`, `e2fsprogs`, `xfsprogs`,
   `btrfs-progs`, `ntfs-3g`/`ntfsprogs`, `hivex`/`perl-hivex`, …), then
   [`build/appliance/pack.sh`](../../build/appliance/pack.sh) copies the kernel
   to `vmlinuz`, builds a trimmed initramfs root containing only the needed
   virtio and filesystem kernel modules plus `kc-agent` at `/kc-agent`, strips
   docs/locales/caches, and packs it as `initramfs.img` (cpio + xz).

## Runtime discovery

The backend locates the artifacts through `KC_APPLIANCE_DIR`:

- If `KC_APPLIANCE_DIR` is set, it must contain `vmlinuz` and `initramfs.img`
  for the host architecture.
- Otherwise the backend looks next to the executable (`<exe-dir>/appliance/<arch>`)
  and falls back to `build/appliance/out/<arch>`.

QEMU is launched with `-kernel vmlinuz -initrd initramfs.img` and
`-append "rdinit=/kc-agent …"`, so `kc-agent` runs as pid 1. Guest disks are
attached as virtio-blk devices; the agent RPC channel is a virtio-serial port
named `org.kc-utils.agent` bound to `KC_AGENT_SOCK`. See
[`pkg/guest/plugins/qemu/cmdline.go`](../../pkg/guest/plugins/qemu/cmdline.go) for the full
argv.
