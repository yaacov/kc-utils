# pkg/guest/plugins/direct — host-kernel backend

Direct backend for guest disk operations using host kernel mounts (losetup,
LVM, cryptsetup, mount/umount). Requires CAP_SYS_ADMIN or a privileged pod.

All domain logic (discovery, mount, LUKS, LVM, fsck, chroot, hive merge) lives
in [`../../core`](../../core/) on top of a host-local
[`../../runtime`](../../runtime/). This package adds only the host-specific
parts: loop-device attachment ([`disk/`](disk/)) and teardown.

See also: [docs/architecture/filesystem-checks.md](../../../../docs/architecture/filesystem-checks.md)
for `FSCheck` command mapping and check-vs-repair semantics.

## Exports

| Symbol | Role |
|--------|------|
| `New()` | Creates an unattached direct backend (core + host-local runtime) |
| `NewMounted()` | Wraps an already-mounted guest (convert/finalize re-attach) |
| `TeardownMountRoot()` | Best-effort orphan cleanup for direct mode |

## Internal packages

| Package | Role |
|---------|------|
| [`disk/`](disk/) | Loop device attach/detach via losetup + partx |

Import path: `github.com/yaacov/kc-utils/pkg/guest/plugins/direct`

Only imported by the parent `pkg/guest` package — never by code outside `pkg/guest/`.
