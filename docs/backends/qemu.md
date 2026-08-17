# QEMU Backend (QEMU appliance)

`--backend=qemu` / `V2V_backend=qemu`. Registered on **Linux and Darwin**.

The qemu backend runs conversion tools (`mount`, LVM, `cryptsetup`, `fsck`,
`hivexregedit`) inside a kc-utils-built QEMU appliance rather than on the host.
Unlike the [guestfs backend](guestfs.md), it does not depend on libguestfs or
supermin — kc-utils launches QEMU directly with a shipped kernel and initramfs
and talks to [`kc-agent`](kc-agent.md) over a virtio-serial Unix socket. This
is the only backend that runs on macOS.

Implementation: [`pkg/guest/plugins/qemu/`](../../pkg/guest/plugins/qemu/) (host
client + remote runtime). Domain logic is shared with `direct` via
[`pkg/guest/core`](../../pkg/guest/core/); the in-VM agent is
[`pkg/agent`](../../pkg/agent/) (Linux-only) with shared RPC types in
[`pkg/agent/protocol`](../../pkg/agent/protocol/). See [kc-agent.md](kc-agent.md).

```text
kc-v2v / kc-prepare / kc-finalize (host, unprivileged)
  └── QEMU (accel: kvm on Linux, hvf on macOS, else tcg)
        ├── kernel + initramfs from KC_APPLIANCE_DIR (see appliance.md)
        ├── rdinit=/kc-agent  (pid 1 inside the VM; see kc-agent.md)
        │     mount, LVM, LUKS, fsck, hivexregedit ...
        ├── virtio-serial (org.kc-utils.agent) ←→ KC_AGENT_SOCK RPC
        └── virtio-serial (org.kc-utils.shell) ←→ sibling shell.sock (kc-agent-sh)
```

## Requirements

The conversion host needs QEMU (`qemu-system-x86_64` or
`qemu-system-aarch64`) and two appliance files from `make appliance`:
`vmlinuz` and `initramfs.img` (found via `KC_APPLIANCE_DIR`). Build them as
described in [appliance.md](appliance.md).

Virtio-win and qemu-guest-agent RPMs stay on the **host**
(`/usr/share/virtio-win`, `/usr/share/kc-packages`, or `KC_VIRTIO_WIN` /
`KC_PACKAGES` from `make stage-offline`) — they are not packed into the
appliance.

Accel is `hvf` on Darwin, `kvm` on Linux when `/dev/kvm` exists, otherwise
`tcg` (software emulation, much slower).

## Environment variables

| Variable | Purpose |
|----------|---------|
| `KC_AGENT_SOCK` | Unix socket for `kc-agent` RPC (set by `kc-v2v` shared session). Debug shell is the sibling `shell.sock`. |
| `KC_QEMU_PID` | QEMU pid after prepare Setup (liveness) |
| `KC_APPLIANCE_DIR` | Directory with `vmlinuz` and `initramfs.img` for host `GOARCH` |
| `KC_VIRTIO_WIN` | Host virtio-win tree (same as other backends; not packed in the appliance) |
| `KC_PACKAGES` | Host qemu-ga package tree (same as other backends) |
| `KC_GUESTFS_NETWORK` | `1`/`true` enables QEMU user-net for Clevis (same env as guestfs) |

`V2V_memSize` / `LIBGUESTFS_MEMSIZE` and `V2V_smp` / `LIBGUESTFS_SMP` tune the
appliance VM (default 2048 MiB, up to 8 vCPUs).

## Shared-session ownership

Shared-session ownership matches guestfs:

1. **`kc-v2v` reserves** a Unix socket path (`KC_AGENT_SOCK`).
2. **prepare Setup starts QEMU** with guest disks and `rdinit=/kc-agent`, then
   writes `qemu.pid`. Convert/finalize dial the agent socket; they do not spawn
   a second VM.
3. **`kc-v2v` kills QEMU** after finalize (`KC_QEMU_PID` / pidfile).

Standalone `kc-prepare` without `KC_AGENT_SOCK` starts a process-local QEMU and
tears it down on `Release`/`Teardown`.

## macOS

Install Homebrew QEMU. virtio-win and qemu-ga RPMs stay on the host
(`KC_VIRTIO_WIN` / `KC_PACKAGES`). Stage those trees with `make stage-offline`
(Fedora container → gitignored `build/offline/`). The Mac does not need hivex,
libguestfs, or LVM — those run inside the appliance via `kc-agent`.

End-to-end local flow (NFC copy → qemu backend → boot the converted x86 guest):
[macos-local.md](../apps/macos-local.md).

## See also

- [appliance.md](appliance.md) — building the `vmlinuz` + `initramfs.img` artifacts
- [kc-agent.md](kc-agent.md) — the in-appliance agent and RPC protocol
- [../apps/kc-agent-sh.md](../apps/kc-agent-sh.md) — interactive debug shell into a running appliance
- [clevis-nbde.md](clevis-nbde.md) — Clevis/NBDE unlock
- [filesystem-checks.md](../architecture/filesystem-checks.md) — `FSCheck` matrix
