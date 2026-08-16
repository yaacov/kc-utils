# kc-agent (in-appliance RPC agent)

`kc-agent` is the guest-side half of the [`qemu` backend](qemu.md). It is a
**Linux-only** binary ([`cmd/kc-agent`](../../cmd/kc-agent/main.go)) that runs
as **pid 1** inside the [QEMU appliance](appliance.md) (`rdinit=/kc-agent`) and
executes all privileged guest disk operations there, so the host never mounts
guest filesystems or needs `CAP_SYS_ADMIN`.

It is not invoked by users directly and has no CLI flags — the qemu backend
starts QEMU with `kc-agent` as its init, and the host client talks to it over
RPC.

## Where the code lives

| Path | Role | Build tag |
|------|------|-----------|
| [`cmd/kc-agent/main.go`](../../cmd/kc-agent/main.go) | Entry point: bootstrap, then serve | `//go:build linux` |
| [`pkg/guest/qemu/server/`](../../pkg/guest/qemu/server/) | In-VM agent implementation (dispatch, mount, LUKS, fsck, discover) | linux only |
| [`pkg/guest/qemu/protocol/`](../../pkg/guest/qemu/protocol/protocol.go) | RPC framing and op/arg/result types | none (shared) |
| [`pkg/guest/qemu/`](../../pkg/guest/qemu/) | Host-side client, session, QEMU cmdline | `//go:build unix` |

The `protocol` package has **no OS build tag** so the host client (unix) and
the in-VM server (linux) share types without importing each other. The host
`pkg/guest/qemu` must **not** import `pkg/guest/qemu/server`.

Built as a static binary: `CGO_ENABLED=0 GOOS=linux go build ./cmd/kc-agent`
(target `make kc-agent`; also built inside `make appliance`).

## Startup (pid 1)

`server.Bootstrap()` prepares a minimal pid-1 environment: it mounts `/proc`,
`/sys`, and `/dev` (devtmpfs), `modprobe`s the virtio and filesystem modules
(`virtio_*`, `ext4`, `xfs`, `btrfs`, `vfat`, `ntfs3`, `dm_mod`, `dm_crypt`),
then opens the virtio-serial port `/dev/virtio-ports/org.kc-utils.agent`. That
port is the RPC channel; on the host it is the `KC_AGENT_SOCK` Unix socket.

## RPC protocol

Framing (see [`protocol.go`](../../pkg/guest/qemu/protocol/protocol.go)):

- Each request/response is a 4-byte big-endian length prefix + JSON
  (`Request` / `Response`), capped at 32 MiB.
- Binary payloads (upload/download/read/write/device I/O, run-command output)
  follow as an 8-byte length-prefixed blob when `Size > 0`.

The agent serves a fixed set of ops keyed by `Op` string, including: `mount` /
`unmount_all`, `discover`, `fstype` / `blkid_attr`, `fscheck` / `fstrim`,
`decrypt` / `unlock_clevis` / `close_crypt`, `run_command`, file ops
(`read_file` / `write_file` / `exists` / `glob` / `read_dir` / `upload` /
`download` / …), `merge_hive` (Windows registry via hivex), and `set_root`.
Guest paths are resolved relative to `/mnt/kc-guest` inside the VM.

`fscheck` maps ext*/xfs/btrfs/ntfs3 to `e2fsck -f -y` / `xfs_repair` /
`btrfs check` / `ntfsfix -d` — the same command matrix as the direct backend
(see [filesystem-checks.md](../architecture/filesystem-checks.md)).

## Lifecycle

The host side owns the VM lifecycle — see
[qemu.md → Shared-session ownership](qemu.md#shared-session-ownership). `kc-v2v`
reserves `KC_AGENT_SOCK`, prepare Setup launches QEMU (and thus `kc-agent`) and
records `KC_QEMU_PID`, and `kc-v2v` kills QEMU after finalize. The agent serves
requests until the socket reaches EOF and the VM is torn down.
