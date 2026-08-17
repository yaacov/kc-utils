# kc-agent (in-appliance RPC agent)

`kc-agent` is the guest-side half of the [`qemu` backend](qemu.md). It is a
**Linux-only** binary ([`cmd/kc-agent`](../../cmd/kc-agent/main.go)) that runs
as **pid 1** inside the [QEMU appliance](appliance.md) (`rdinit=/kc-agent`).

It is a **generic runtime**, not a disk-operations service: it only runs
commands and performs raw file/device I/O on absolute paths in its own
namespace. It holds **no domain state or logic** — no mount/LUKS/LVM
bookkeeping, no filesystem detection, no fsck matrix. All of that lives
host-side in [`pkg/guest/core`](../../pkg/guest/core/), which drives the agent
by running the standard tools (`lsblk`, `mount`, `cryptsetup`, `vgchange`,
`e2fsck`, …) over the `exec` op. Those privileged tools still execute inside the
VM, so the host never mounts guest filesystems or needs `CAP_SYS_ADMIN`.

This is the same domain code the `direct` backend runs on the host — `core` sits
on a [`runtime`](../../pkg/guest/runtime/) transport that is host-local for
`direct` and this RPC for `qemu` (see [qemu.md](qemu.md)).

It is not invoked by users directly and has no CLI flags — the qemu backend
starts QEMU with `kc-agent` as its init, and the host client talks to it over
RPC. For an interactive debug shell into a running appliance, use
[`kc-agent-sh`](../apps/kc-agent-sh.md); that helper uses a **second**
virtio-serial port and does not take over the RPC socket.

## Where the code lives

| Path | Role | Build tag |
|------|------|-----------|
| [`cmd/kc-agent/main.go`](../../cmd/kc-agent/main.go) | Thin entry point | `//go:build linux` |
| [`pkg/cmd/agent/`](../../pkg/cmd/agent/) | Orchestrator: `Run()` = bootstrap + serve | `//go:build linux` |
| [`pkg/agent/`](../../pkg/agent/) | Generic runtime: primitive-op dispatch (exec, file/device I/O, stat) | linux only |
| [`pkg/agent/protocol/`](../../pkg/agent/protocol/protocol.go) | RPC framing and op/arg/result types | none (shared) |
| [`pkg/guest/plugins/qemu/`](../../pkg/guest/plugins/qemu/) | Host-side client, remote runtime, session, QEMU cmdline | `//go:build unix` |
| [`cmd/kc-agent-sh/main.go`](../../cmd/kc-agent-sh/main.go) | Host debug-shell helper | `//go:build unix` |
| [`pkg/cmd/agentsh/`](../../pkg/cmd/agentsh/) | Dial shell.sock, raw TTY, copy | `//go:build unix` |

The `protocol` package has **no OS build tag** so the host client (unix) and
the in-VM agent (linux) share types without importing each other. The host
`pkg/guest/plugins/qemu` must **not** import `pkg/agent` (only `pkg/agent/protocol`).

Built as a static binary: `CGO_ENABLED=0 GOOS=linux go build ./cmd/kc-agent`
(target `make kc-agent`; also built inside `make appliance`).

## Startup (pid 1)

`agent.Bootstrap()` prepares a minimal pid-1 environment: it mounts `/proc`,
`/sys`, and `/dev` (devtmpfs), `modprobe`s the virtio and filesystem modules
(`virtio_*`, `ext4`, `xfs`, `btrfs`, `vfat`, `ntfs3`, `dm_mod`, `dm_crypt`),
then opens the virtio-serial port `/dev/virtio-ports/org.kc-utils.agent`. That
port is the RPC channel; on the host it is the `KC_AGENT_SOCK` Unix socket.
A second goroutine opens `/dev/virtio-ports/org.kc-utils.shell` (best-effort)
and serves a PTY bash whenever a host client is connected.

## Debug shell

[`kc-agent-sh`](../apps/kc-agent-sh.md) attaches while conversion RPC is in
use. QEMU binds a sibling Unix socket `shell.sock` next to `agent.sock`
(`protocol.ShellSock`) to virtio-serial port `org.kc-utils.shell`. The agent
starts `/bin/bash` on a PTY when sysfs `host_connected` is 1, and tears the
session down when the host disconnects.

The helper writes a one-line JSON [`ShellConfig`](../../pkg/agent/protocol/protocol.go)
header (`chroot`, `argv`, `term`, window size) then copies raw PTY bytes. An
empty object starts interactive bash in the appliance namespace. This channel
is not part of the RPC opcode set.

Rebuild the appliance (`make appliance`) so pid 1 includes the shell listener.

## RPC protocol

Framing (see [`protocol.go`](../../pkg/agent/protocol/protocol.go)):

- Each request/response is a 4-byte big-endian length prefix + JSON
  (`Request` / `Response`), capped at 32 MiB.
- Binary payloads (file read/write, device I/O) follow as an 8-byte
  length-prefixed blob when `Size > 0`.

The agent serves a small, fixed set of **primitive** ops keyed by `Op` string:
`ping`, `exec` (run a command; stdout/stderr/exit returned inline, a non-zero
exit is not an RPC error), file ops (`read_file` / `write_file` / `mkdir_all` /
`remove` / `remove_all` / `rename` / `symlink` / `readlink` / `chmod` / `stat` /
`read_dir` / `glob`), raw `device_read` / `device_write`, and `statfs`. There are
**no** domain ops — mount, decrypt, discover, fsck, chroot, and hive-merge are
all composed host-side in `core` out of `exec` + file I/O.

Because the agent has no notion of a guest root, **all paths are absolute in the
VM's namespace**. Guest-to-host path translation (e.g. `/etc/fstab` →
`<mountRoot>/etc/fstab`) happens host-side in `core`; the agent receives the
already-resolved absolute path.

## Lifecycle

The host side owns the VM lifecycle — see
[qemu.md → Shared-session ownership](qemu.md#shared-session-ownership). `kc-v2v`
reserves `KC_AGENT_SOCK`, prepare Setup launches QEMU (and thus `kc-agent`) and
records `KC_QEMU_PID`, and `kc-v2v` kills QEMU after finalize. The agent serves
requests until the socket reaches EOF and the VM is torn down.
