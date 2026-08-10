# pkg/guest/direct — host-kernel backend

Direct backend for guest disk operations using host kernel mounts (losetup,
LVM, cryptsetup, mount/umount). Requires CAP_SYS_ADMIN or a privileged pod.

See also: [docs/architecture/filesystem-checks.md](../../../docs/architecture/filesystem-checks.md)
for `FSCheck` command mapping and check-vs-repair semantics.

## Exports

| Symbol | Role |
|--------|------|
| `New()` | Creates an unattached direct backend |
| `NewMounted()` | Wraps an already-mounted guest (convert/finalize re-attach) |
| `TeardownMountRoot()` | Best-effort orphan cleanup for direct mode |

## Internal packages

| Package | Role |
|---------|------|
| [`disk/`](disk/) | Loop device attach/detach, partition scanning |
| [`luks/`](luks/) | LUKS open/close via cryptsetup and Clevis |
| [`lvm/`](lvm/) | LVM volume group activation/deactivation |
| [`mount/`](mount/) | Mount/unmount, remount, /proc/mounts parsing |
| [`fstype/`](fstype/) | Filesystem type detection via `blkid` |

Import path: `github.com/yaacov/kc-utils/pkg/guest/direct`

Only imported by the parent `pkg/guest` package — never by code outside `pkg/guest/`.
