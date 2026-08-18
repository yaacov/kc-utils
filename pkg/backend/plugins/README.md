# backend plugins

Pluggable guest disk backends registered into `backend.Plugins`.

| Key | Package | Requirements | Description |
|-----|---------|--------------|-------------|
| `direct` | direct/ | Linux, root/CAP_SYS_ADMIN | Host kernel mounts via losetup, LVM, cryptsetup |
| `guestfs` | guestfs/ | Linux, `/dev/kvm` | libguestfs appliance via `guestfish --listen` |

Selection is explicit via `--backend direct|guestfs` (default `direct`). Runtime probes in `pkg/backend/runtime.go` gate startup.

Import path: `github.com/yaacov/kc-utils/pkg/backend/plugins/<name>`
