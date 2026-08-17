# pkg/guest/plugins/qemu — QEMU appliance backend

QEMU backend for guest disk operations. Boots a minimal appliance VM (shipped
kernel + initramfs) with the guest disks attached as virtio-blk devices, and
drives it from the host over an RPC to `kc-agent` running inside the VM. Runs
where `direct` cannot (no CAP_SYS_ADMIN) and where `guestfs` is unavailable;
supported on Linux and Darwin.

## Architecture

All domain logic (discovery, mount, LUKS, LVM, fsck, chroot, hive merge) lives
in [`../../core`](../../core/). This package supplies a **remote runtime**
([`remote.go`](remote.go)) that forwards `core`'s primitive operations (exec,
file/device I/O, stat) to `kc-agent` over the RPC client, so the exact same
domain code runs whether the runtime is host-local (`direct`) or in the VM here.

The appliance agent is a generic runtime with no domain knowledge — it only runs
commands and reads/writes files and devices on absolute paths. It lives in
[`pkg/agent`](../../../agent/) (served by `cmd/kc-agent`); the wire protocol is
[`pkg/agent/protocol`](../../../agent/protocol/). QEMU also binds a second
virtio-serial port (`org.kc-utils.shell` / sibling `shell.sock`) for the
[`kc-agent-sh`](../../../../docs/apps/kc-agent-sh.md) debug PTY.

Disks get a `serial=kc-disk-<index>` on the QEMU command line
([`cmdline.go`](cmdline.go)); `discover` reads those serials back via `lsblk` to
map appliance devices to disk specs by index.

## Files

| File | Role |
|------|------|
| [`backend.go`](backend.go) | `Backend` (embeds `core.Backend`), Setup/discover, lifecycle |
| [`remote.go`](remote.go) | `runtime.Runtime` over the RPC client |
| [`client.go`](client.go) | Length-prefixed JSON + blob RPC client |
| [`session.go`](session.go) | Shared session, QEMU launch, agent wait |
| [`cmdline.go`](cmdline.go) | `BuildQEMUArgs`, appliance launch config |
| [`register.go`](register.go) | Factory registration into `backend.Factories` |

Import path: `github.com/yaacov/kc-utils/pkg/guest/plugins/qemu`

Only imported by the parent `pkg/guest` package (blank import from `cmd/`
mains) — never by code outside `pkg/guest/`. Must not import `pkg/agent`
(the linux-only appliance server); only `pkg/agent/protocol` is shared.
