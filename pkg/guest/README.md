# pkg/guest — privileged guest disk operations

Sole boundary for privileged host, libguestfs, and QEMU-appliance operations
on guest disks. Callers must not invoke guestfish, mount/umount, losetup, LVM,
cryptsetup, chroot, fsck, or fstrim for guest disks outside this package.

See also: [docs/architecture/filesystem-checks.md](../../docs/architecture/filesystem-checks.md)
for fsck timing, supported filesystem types, and check-vs-repair behavior, and
[docs/backends/](../../docs/backends/README.md) for the user-facing backend docs
(selection, privilege model, the QEMU appliance, and `kc-agent`).

## Backend plugins

Backends self-register into `guest.Factories` (`plugin.Registry`) via `init()`.
Stage binaries blank-import the implementations they ship:

| Name | Package | Mechanism | Requirements |
|------|---------|-----------|--------------|
| `direct` | [`direct/`](direct/) | Host kernel mounts via losetup, LVM, cryptsetup | CAP_SYS_ADMIN / privileged pod; registered on Linux only |
| `guestfs` | [`guestfs/`](guestfs/) | libguestfs appliance RPC via `guestfish --listen` | `/dev/kvm`, unprivileged pod; registered on Linux only |
| `qemu` | [`qemu/`](qemu/) | QEMU + shipped kernel/initramfs; `kc-agent` RPC | QEMU + appliance files; registered on Linux and Darwin |

Select with `--backend <name>` / `V2V_backend` (**required**; runtime list from `Factories.List()`).
There is no default. The `kc-v2v` image sets `V2V_backend=guestfs` explicitly.

All backends implement the `Backend` interface. The `Guest` facade normalizes
guest paths and delegates to the active backend — the rest of the codebase is
unaware of which backend is in use.

`direct` and `guestfs` compile on Unix but call `Factories.Register` only when
`runtime.GOOS == "linux"`. Darwin `Factories` lists `qemu` only.

### Isolation rules

- Backend packages must not import each other or share mutable package state.
- Parent `pkg/guest` must not import concrete backends (blank imports in `cmd/`).
- Shared helpers live in [`common/`](common/) (host copy/`StatFS` only).
- Stages/blocks use the `Guest` handle — never `pkg/guest/common` for disk I/O.
- Host `pkg/guest/qemu` must not import `pkg/guest/qemu/server` (linux-only). Shared RPC types live in [`qemu/protocol/`](qemu/protocol/).

### Adding a backend

1. New package under `pkg/guest/<name>/` implementing `guest.Backend` + `guest.Factory`
2. Optional: `SharedSessionFactory`, `ClevisAwareFactory`
3. `init() { guest.Factories.Register("<name>", …) }` (linux-only backends should return unless `runtime.GOOS == "linux"`)
4. Blank-import from stage / `kc-v2v` mains

## Key types and functions

| Symbol | Role |
|--------|------|
| `Guest` | High-level handle used by prepare/convert/finalize pipelines |
| `Backend` | Interface satisfied by backend plugins |
| `Factory` / `Factories` | Registry of named backend factories |
| `Mode` | Registry key string (`direct`, `guestfs`, `qemu`, …) |
| `Open()` | Looks up factory, creates backend, runs Setup |
| `AttachMounted()` | Reconnects to an already-prepared guest (convert/finalize entry) |
| `TeardownMountRoot()` | Best-effort orphan cleanup when handoff data is unavailable |
| `SharedSession` | Cross-stage session (guestfs `guestfish --listen`; qemu `KC_AGENT_SOCK`) |
| `StartSharedSession()` | Starts a session when the backend implements `SharedSessionFactory` |
| `AttachFromPrepare()` | Convenience wrapper: derives mode, orders disks, attaches, sets active handle |
| `SetActive()` / `ClearActive()` | Global guest handle for `File*` convenience helpers |
| `FSCheck()` | Filesystem check/repair on unmounted block devices (see architecture doc) |
| `FSTrim()` | Trim mounted guest filesystems (finalize) |
| `UnmountFilesystems()` | Unmount guest FS; keep LUKS/LVM open (finalize, before post-fsck) |
| `ReleaseDevices()` | Close LUKS, deactivate LVM, detach loops (finalize, after post-fsck) |

## File layout

```
pkg/guest/
  guest.go          — Guest facade, Open, AttachMounted
  backend.go        — Backend interface, DirEntry alias
  factory.go        — Factory registry and lookup
  mode.go           — Mode string keys, ParseMode
  listener.go       — SharedSession helpers and env consts
  teardown.go       — TeardownMountRoot via factory
  active.go         — Global active guest handle
  path.go / file_ops.go / checkout.go — Guest path and File* helpers
  common/           — Shared host FS helpers (CopyFile, HostStatFS)
  direct/           — Direct backend plugin
  guestfs/          — Guestfs backend plugin
  qemu/             — QEMU appliance backend (host client + protocol; server is linux-only)
```

Import path: `github.com/yaacov/kc-utils/pkg/guest`
