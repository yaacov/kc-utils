# pkg/guest — privileged guest disk operations

Sole boundary for privileged host and libguestfs operations on guest disks.
Callers must not invoke guestfish, mount/umount, losetup, LVM, cryptsetup,
chroot, fsck, or fstrim for guest disks outside this package.

See also: [docs/architecture/filesystem-checks.md](../../docs/architecture/filesystem-checks.md)
for fsck timing, supported filesystem types, and check-vs-repair behavior.

## Backends

| Mode | Subpackage | Mechanism | Requirements |
|------|------------|-----------|--------------|
| `ModeDirect` | [`direct/`](direct/) | Host kernel mounts via losetup, LVM, cryptsetup | CAP_SYS_ADMIN / privileged pod |
| `ModeGuestfs` | [`guestfs/`](guestfs/) | libguestfs appliance RPC via `guestfish --listen` | /dev/kvm, unprivileged pod |

Both backends implement the `Backend` interface (34 methods covering setup,
mount, filesystem ops, encryption, device I/O, and teardown). The `Guest`
facade normalizes guest paths and delegates to the active backend — the rest
of the codebase is unaware of which backend is in use.

## Key types and functions

| Symbol | Role |
|--------|------|
| `Guest` | High-level handle used by prepare/convert/finalize pipelines |
| `Backend` | Interface satisfied by both backends |
| `Mode` | `ModeDirect` or `ModeGuestfs` |
| `Open()` | Factory: creates backend, runs Setup |
| `AttachMounted()` | Reconnects to an already-prepared guest (convert/finalize entry) |
| `TeardownMountRoot()` | Best-effort orphan cleanup when handoff data is unavailable |
| `SharedListener` | Cross-stage guestfish session (re-exported from `guestfs/`) |
| `StartSharedListener()` | Launches the shared listener (re-exported from `guestfs/`) |
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
  mode.go           — Mode enum
  listener.go       — Re-exports SharedListener from guestfs/
  teardown.go       — TeardownMountRoot dispatcher
  active.go         — Global active guest handle
  path.go           — File* convenience helpers (FileRead, FileWrite, …)
  file_ops.go       — Checkout / Checkin (host↔guest file transfer)
  checkout.go       — Extended checkout helpers
  mount_table.go    — /proc/mounts parser
  guest_util.go     — normalizeGuestPath, copyFile, copyDir, hostStatFS
  direct/           — Direct backend subpackage
  guestfs/          — Guestfs backend subpackage
```

Import path: `github.com/yaacov/kc-utils/pkg/guest`
