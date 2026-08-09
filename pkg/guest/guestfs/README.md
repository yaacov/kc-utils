# pkg/guest/guestfs — libguestfs backend

Guestfs backend for guest disk operations using a libguestfs appliance.
Guest filesystems stay inside the appliance VM; all I/O uses guestfish RPC.
Suitable for unprivileged pods with /dev/kvm access.

## Exports

| Symbol | Role |
|--------|------|
| `New()` | Creates an unattached guestfs backend |
| `NewMounted()` | Attaches to a running guestfish session (convert/finalize re-attach) |
| `TeardownMountRoot()` | Best-effort orphan cleanup for guestfs mode |
| `SharedListener` | Long-lived guestfish session shared across pipeline stages |
| `StartSharedListener()` | Launches `virt-guestfish --listen` when present (else `guestfish`) |
| `EnvGuestfishPID` | Environment variable name for the guestfish PID |
| `EnvKCGuestfishPID` | Environment variable name for kc-managed guestfish PID |

## File layout

```
backend.go      — Backend struct, New, NewMounted, TeardownMountRoot
fs.go           — Filesystem operations (read/write/glob/readdir/upload/download)
session.go      — guestfish --listen session management, SharedListener
discover.go     — Partition and LVM discovery via guestfish
probe.go        — Filesystem probe mount/unmount
mount_specs.go  — Mount spec recording for session re-attach; inspect-os
                  enrichment matches preferred `/` (else first); path conflicts
                  keep prepare
util.go         — guestfishBinary, quoteGuestfish, runGuestfsCmd, scripts
```

On RHEL/UBI, prefer `virt-guestfish` when present (symlink) so NTFS mounts pass
the winsupport allowlist; Fedora uses plain `guestfish` — see
[docs/privilege-model.md](../../../docs/privilege-model.md#ntfs-mounts-on-rhelcentosubi).

Import path: `github.com/yaacov/kc-utils/pkg/guest/guestfs`

Only imported by the parent `pkg/guest` package — never by code outside `pkg/guest/`.
