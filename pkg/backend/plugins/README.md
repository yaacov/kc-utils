# backend plugins

Pluggable guest disk backends registered into `backend.Plugins`.

| Key | Package | Requirements | Description |
|-----|---------|--------------|-------------|
| `direct` | direct/ | Linux, root/CAP_SYS_ADMIN | Host kernel mounts via losetup, LVM, cryptsetup |
| `guestfs` | guestfs/ | Linux, `/dev/kvm` | libguestfs appliance via `guestfish --listen` |
| `qemu` | qemu/ | Linux **or** macOS, `qemu-system-*` (KVM/HVF/TCG) + appliance image for the selected arch (`KC_APPLIANCE_ARCH`, default host `GOARCH`) | Own minimal appliance booted directly with qemu; primitive agent over a unix socket |

Selection is explicit via `--backend direct|guestfs|qemu` (default `direct`). Runtime probes in `pkg/backend/runtime.go` gate startup.

The `qemu` backend is the only one that also runs on macOS: the host side only
launches `qemu-system-*` and speaks a unix socket, while all Linux tooling runs
inside the appliance. See [qemu/README.md](qemu/README.md) and
[docs/architecture/qemu-appliance.md](../../../docs/architecture/qemu-appliance.md).

Import path: `github.com/yaacov/kc-utils/pkg/backend/plugins/<name>`
