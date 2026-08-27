# kc-guest-agent

In-appliance PID 1 for the `qemu` guest-disk backend. Serves primitive
operations over a virtio-serial port so the **host** can compose prepare /
convert / finalize. This binary is not a pipeline stage and is not run on the
conversion host.

This is **not** [qemu-guest-agent](kc-convert-linux.md#qemu-guest-agent-installation)
(the package converters inject into the converted guest). `kc-guest-agent`
lives only inside the kc appliance initramfs.

Runs [`pkg/qemuagent/server`](../../pkg/qemuagent/) over the wire contract in
[`pkg/qemuagent/proto`](../../pkg/qemuagent/proto/).

**Entry:** `cmd/kc-guest-agent/main.go` — `--port` and `--init`; no
subcommands. PID-1 bootstrap and the debug bash channel live in
`bootstrap_linux.go` and `channel_linux.go`.

Compiles on any Unix so `go build ./...` works on a Mac. At runtime it only
serves inside a Linux appliance (`bootstrap_other.go` refuses to run on other
GOOS).

## CLI Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--port` | no | `/dev/virtio-ports/org.kc-utils.agent` | Virtio-serial device to serve the framed protocol on |
| `--init` | no | `false` (forced on when PID is 1) | Act as PID 1: mount core filesystems, load virtio modules, start the debug channel, then serve |

Exit codes: `0` success, `1` serve or init error. As PID 1 the process never
exits: after a host disconnect it reopens the port and keeps serving.

## What `--init` does

When the binary is `/init` (or `--init` is set):

1. Sets `PATH=/usr/sbin:/usr/bin:/sbin:/bin` if unset (PID 1 inherits none).
2. Mounts `/proc`, `/sys`, `/dev` (devtmpfs), `/dev/pts` (needed for the debug
   PTY), `/run`, `/tmp`. Failures are non-fatal (already mounted).
3. `modprobe`s virtio drivers (`virtio`, `virtio_pci`, `virtio_console`,
   `virtio_blk`, `virtio_scsi`, `virtio_net`). Failures are non-fatal.
4. If `qemu-*-static` is packaged, mounts `binfmt_misc` and registers those
   interpreters with the **F** (fix-binary) flag so `chroot` can exec a guest
   ELF whose ISA is not the appliance's. Missing interpreters are a no-op;
   packaged qemu-user that fails to register is a fatal init error (otherwise
   later guest commands are `Exec format error`).
5. Waits for a host client on virtio-serial port `org.kc-utils.debug`, then
   starts `/bin/bash -i` on a PTY. Independent of the JSON protocol. Bash is
   not spawned until something connects to `debug.sock`; after disconnect it
   waits for the next client.
6. Serves the agent protocol on `--port`, reopening after each clean
   disconnect so pipeline stages can reconnect.

The named node `/dev/virtio-ports/<name>` is a udev artifact. With no udevd
the agent falls back to `/sys/class/virtio-ports/*/name` and opens the raw
`/dev/vportNpM` device.

## Primitive operations

Length-prefixed JSON frames (4-byte big-endian length + one JSON object).
`[]byte` fields are base64. One flat `Request`, one flat `Response`. Framing
and field layout: [backends.md](../architecture/backends.md#wire-protocol).

| Op | Role |
|----|------|
| `ping` | Liveness; host polls this until boot is ready |
| `exec` | Run an appliance executable; returns stdout/stderr/exit |
| `readfile` / `writefile` | Whole-file I/O |
| `stat` / `readdir` / `statfs` | Metadata |
| `mkdir` / `remove` / `rename` / `chmod` | Path mutations |
| `symlink` / `readlink` | Symlinks |
| `pread` / `pwrite` | Raw device or file I/O at an offset |

The agent never imports `pkg/backend`. Partition discovery, LVM, LUKS, mount
planning, and conversion logic stay host-side.

## Debug channel

Port name `org.kc-utils.debug`, bridged by QEMU to `debug.sock` next to the
agent socket. Attach from the host with `socat`.
You are in the **appliance** initramfs, not the converted guest. After prepare
mounts filesystems they appear under `/mnt/guest`.

How-to (attach snippet, held appliance, per-stage checks):
[../debug/README.md](../debug/README.md). Design notes:
[backends.md — Interactive debug shell](../architecture/backends.md#interactive-debug-shell).

## Integration

The appliance build ([`build/kc-appliance`](../../build/kc-appliance)) compiles
this binary as `/init` inside `initramfs.img`. Rebuild with
`make build-appliance` after agent changes; host-side qemu launch changes
alone are not enough.

The host qemu backend locates `vmlinuz` + `initramfs.img` under
`KC_APPLIANCE_DIR` (default `/usr/lib/kc-utils/appliance`) /
`KC_APPLIANCE_ARCH` (default host `GOARCH`). The agent is **not** shipped in
the `kc-v2v` container image (that image converts with `guestfs`).

## Related

- [../architecture/backends.md](../architecture/backends.md) — guest disk backends (host/guest split and protocol)
- [../../pkg/qemuagent/README.md](../../pkg/qemuagent/README.md) — `proto/` and `server/` packages
- [../../build/kc-appliance/README.md](../../build/kc-appliance/README.md) — kernel + initramfs build
- [../debug/start-appliance.md](../debug/start-appliance.md) — boot the appliance and attach the debug shell
