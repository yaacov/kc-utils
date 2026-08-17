# pkg/guest — privileged guest disk operations

Sole boundary for privileged host, libguestfs, and QEMU-appliance operations
on guest disks. Callers must not invoke guestfish, mount/umount, losetup, LVM,
cryptsetup, chroot, fsck, or fstrim for guest disks outside this package.

See also: [docs/architecture/filesystem-checks.md](../../docs/architecture/filesystem-checks.md)
for fsck timing, supported filesystem types, and check-vs-repair behavior, and
[docs/backends/](../../docs/backends/README.md) for the user-facing backend docs
(selection, privilege model, the QEMU appliance, and `kc-agent`).

## Backend plugins

Backends self-register into `backend.Factories` (`plugin.Registry`) via `init()`.
The plugin contract (`Backend` interface, `Factory`, `Mode`, `SharedSession`)
lives in the [`backend/`](backend/) sub-package, so plugins depend on the
contract without importing the `Guest` facade. Stage binaries blank-import the
implementations they ship:

| Name | Package | Mechanism | Requirements |
|------|---------|-----------|--------------|
| `direct` | [`plugins/direct/`](plugins/direct/) | Host kernel mounts via losetup, LVM, cryptsetup | CAP_SYS_ADMIN / privileged pod; registered on Linux only |
| `guestfs` | [`plugins/guestfs/`](plugins/guestfs/) | libguestfs appliance RPC via `guestfish --listen` | `/dev/kvm`, unprivileged pod; registered on Linux only |
| `qemu` | [`plugins/qemu/`](plugins/qemu/) | QEMU + shipped kernel/initramfs; `kc-agent` RPC | QEMU + appliance files; registered on Linux and Darwin |

See [`plugins/README.md`](plugins/README.md) for the shared plugin contract.

Select with `--backend <name>` / `V2V_backend` (**required**; runtime list from `backend.Factories.List()`).
There is no default. The `kc-v2v` image sets `V2V_backend=guestfs` explicitly.

All backends implement the `backend.Backend` interface. The `Guest` facade
normalizes guest paths and delegates to the active backend — the rest of the
codebase is unaware of which backend is in use.

`direct` and `guestfs` compile on Unix but call `backend.Factories.Register` only
when `runtime.GOOS == "linux"`. Darwin `backend.Factories` lists `qemu` only.

### Isolation rules

- Backend packages must not import each other or share mutable package state.
- Parent `pkg/guest` must not import concrete backends (blank imports in `cmd/`).
- Domain logic shared by `direct` and `qemu` lives in [`core/`](core/) on top of a
  [`runtime/`](runtime/) transport (host-local or RPC); backends add only device
  attachment and process lifecycle.
- Shared helpers live in [`common/`](common/) (host copy/`StatFS` only).
- Stages/blocks use the `Guest` handle — never `pkg/guest/common` for disk I/O.
- Host `plugins/qemu` must not import the linux-only agent (`pkg/agent`). Shared
  RPC types live in [`pkg/agent/protocol`](../agent/protocol/).

### Adding a backend

1. New package under `pkg/guest/plugins/<name>/` implementing `backend.Backend` + `backend.Factory`
2. Optional: `backend.SharedSessionFactory`, `backend.ClevisAwareFactory`
3. `init() { backend.Factories.Register("<name>", …) }` (linux-only backends should return unless `runtime.GOOS == "linux"`)
4. Blank-import from stage / `kc-v2v` mains

## Key types and functions

Facade symbols live in `pkg/guest`; the plugin contract lives in
`pkg/guest/backend` (imported as `backend`).

| Symbol | Pkg | Role |
|--------|-----|------|
| `Guest` | guest | High-level handle used by prepare/convert/finalize pipelines |
| `backend.Backend` | backend | Interface satisfied by backend plugins |
| `backend.Factory` / `backend.Factories` | backend | Registry of named backend factories |
| `backend.Mode` | backend | Registry key string (`direct`, `guestfs`, `qemu`, …) |
| `backend.SharedSession` | backend | Cross-stage session (guestfs `guestfish --listen`; qemu `KC_AGENT_SOCK`) |
| `backend.StartSharedSession()` | backend | Starts a session when the backend implements `SharedSessionFactory` |
| `Open()` | guest | Looks up factory, creates backend, runs Setup |
| `AttachMounted()` | guest | Reconnects to an already-prepared guest (convert/finalize entry) |
| `TeardownMountRoot()` | guest | Best-effort orphan cleanup when handoff data is unavailable |
| `AttachFromPrepare()` | guest | Convenience wrapper: derives mode, orders disks, attaches, sets active handle |
| `SetActive()` / `ClearActive()` | guest | Global guest handle read by the `guestio.File*` helpers |
| `guestio.File*` | guestio | Path-resolving file ops that route to the active guest or the host FS |
| `FSCheck()` | guest | Filesystem check/repair on unmounted block devices (see architecture doc) |
| `FSTrim()` | guest | Trim mounted guest filesystems (finalize) |
| `UnmountFilesystems()` | guest | Unmount guest FS; keep LUKS/LVM open (finalize, before post-fsck) |
| `ReleaseDevices()` | guest | Close LUKS, deactivate LVM, detach loops (finalize, after post-fsck) |

## File layout

```
pkg/guest/
  guest.go          — Guest facade, Open, AttachMounted
  teardown.go       — TeardownMountRoot via factory
  active.go         — Global active guest handle
  checkout.go       — Guest path helpers used by the facade
  guestio/          — File* convenience layer (host-style paths → guest or host FS)
    file_ops.go     — FileRead/Write/Glob/WalkDir/… routing via the active guest
    path.go         — guestPathFromHost resolution
  backend/          — Plugin contract (imported by facade + plugins)
    backend.go      — Backend interface, DirEntry alias
    factory.go      — Factory registry and lookup
    mode.go         — Mode string keys, ParseMode
    listener.go     — SharedSession helpers and env consts
  common/           — Shared host FS helpers (CopyFile, HostStatFS)
  runtime/          — Domain-free exec + file/device transport (local; remote impl in plugins/qemu)
  core/             — Shared domain logic over a runtime (used by direct + qemu)
  plugins/          — Backend plugins (self-register into backend.Factories)
    direct/         — Direct backend: host loop mounts (core + local runtime)
    guestfs/        — Guestfs backend: libguestfs appliance RPC
    qemu/           — QEMU appliance backend (core + remote runtime → kc-agent)
```

The `kc-agent` in-appliance runtime lives in [`pkg/agent`](../agent/) (served by
`cmd/kc-agent`); its wire protocol is [`pkg/agent/protocol`](../agent/protocol/).

Import path: `github.com/yaacov/kc-utils/pkg/guest`
