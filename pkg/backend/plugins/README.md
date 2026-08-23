# backend plugins

Pluggable guest disk backends registered into `backend.Plugins`.

| Key | Package | Requirements | Description |
|-----|---------|--------------|-------------|
| `direct` | direct/ | Linux, root/CAP_SYS_ADMIN | Host kernel mounts via losetup, LVM, cryptsetup |
| `guestfs` | guestfs/ | Linux, `/dev/kvm`, `guestfish` | libguestfs appliance via `guestfish --listen` |
| `qemu` | qemu/ | `qemu-system-*` + appliance image (`KC_APPLIANCE_ARCH`, default host `GOARCH`) | QEMU appliance with in-guest agent over a unix socket |

Selection is explicit via `--backend direct|guestfs|qemu` (default `direct`). Runtime probes in `pkg/backend/runtime.go` gate startup.

See [qemu/README.md](qemu/README.md) and
[docs/architecture/backends.md](../../../docs/architecture/backends.md).

Import path: `github.com/yaacov/kc-utils/pkg/backend/plugins/<name>`
