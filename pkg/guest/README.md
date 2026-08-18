# pkg/guest — privileged guest disk operations

Sole boundary for privileged host and libguestfs operations on guest disks.
Callers must not invoke guestfish, mount/umount, losetup, LVM, cryptsetup,
chroot, fsck, or fstrim for guest disks outside this package.

Backend implementations live in [`pkg/backend/plugins/`](../backend/plugins/).
Selection uses `--backend direct|guestfs` (default `direct`) with runtime
requirement checks in [`pkg/backend/`](../backend/).

See also: [docs/architecture/filesystem-checks.md](../../docs/architecture/filesystem-checks.md)
for fsck timing, supported filesystem types, and check-vs-repair behavior.

## Backends

| Backend | Plugin | Mechanism | Requirements |
|---------|--------|-----------|--------------|
| `direct` | [`plugins/direct/`](../backend/plugins/direct/) | Host kernel mounts via losetup, LVM, cryptsetup | Linux, CAP_SYS_ADMIN / privileged pod |
| `guestfs` | [`plugins/guestfs/`](../backend/plugins/guestfs/) | libguestfs appliance RPC via `guestfish --listen` | Linux, `/dev/kvm` |

Both backends implement the `Backend` interface in [`pkg/backend/backend.go`](../backend/backend.go).
The `Guest` facade normalizes guest paths and delegates to the active backend — the rest
of the codebase is unaware of which backend is in use.

## Key types and functions

| Symbol | Role |
|--------|------|
| `Guest` | High-level handle used by prepare/convert/finalize pipelines |
| `Backend` | Re-export of `backend.Backend` interface |
| `Open()` | Factory: resolves backend plugin, runs Setup |
| `AttachMounted()` | Reconnects to an already-prepared guest (convert/finalize entry) |
| `TeardownMountRoot()` | Best-effort orphan cleanup when handoff data is unavailable |
| `SharedListener` | Cross-stage guestfish session (guestfs backend only) |
| `StartSharedListener()` | Launches the shared listener via guestfs plugin |
| `AttachFromPrepare()` | Convenience wrapper: orders disks, attaches, sets active handle |
| `SetActive()` / `ClearActive()` | Global guest handle for `File*` convenience helpers |
| `FSCheck()` | Filesystem check/repair on unmounted block devices (see architecture doc) |
| `FSTrim()` | Trim mounted guest filesystems (finalize) |
| `UnmountFilesystems()` | Unmount guest FS; keep LUKS/LVM open (finalize, before post-fsck) |
| `ReleaseDevices()` | Close LUKS, deactivate LVM, detach loops (finalize, after post-fsck) |

## File layout

```
pkg/guest/
  guest.go          — Guest facade, Open, AttachMounted
  backend.go        — Backend type re-exports and backend name constants
  listener.go       — SharedListener via guestfs plugin
  teardown.go       — TeardownMountRoot via backend registry
  active.go           — Global active guest handle
  path.go             — File* convenience helpers (FileRead, FileWrite, …)
  file_ops.go         — Checkout / Checkin (host↔guest file transfer)
  checkout.go         — Extended checkout helpers
  mount_table.go      — /proc/mounts parser
  guest_util.go       — normalizeGuestPath, copyFile, copyDir, hostStatFS
pkg/backend/plugins/
  direct/             — Direct backend plugin
  guestfs/            — Guestfs backend plugin
```

Import path: `github.com/yaacov/kc-utils/pkg/guest`
