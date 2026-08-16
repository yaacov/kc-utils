# guest backend plugins

`backend.Backend` implementations — the swappable engines behind the `Guest`
facade. Each plugin self-registers into `backend.Factories` (a `plugin.Registry`)
from an `init()` function and is wired into a binary by a blank import in the
`cmd/*/main.go` files. Plugins depend on the [`../backend`](../backend/) contract
package, not the `Guest` facade; the parent [`pkg/guest`](../) never imports these
packages directly, so the rest of the codebase stays unaware of which backend
is active.

Select a backend at runtime with `--backend <name>` / `V2V_backend` (**required**;
the live list comes from `backend.Factories.List()`). There is no default.

| Key | Package | Mechanism | Requirements |
|-----|---------|-----------|--------------|
| `direct` | [`direct/`](direct/) | Host kernel mounts via losetup, LVM, cryptsetup | CAP_SYS_ADMIN / privileged pod; Linux only |
| `guestfs` | [`guestfs/`](guestfs/) | libguestfs appliance RPC (`guestfish --listen`) | `/dev/kvm`, unprivileged pod; Linux only |
| `qemu` | [`qemu/`](qemu/) | QEMU appliance + `kc-agent` RPC | QEMU + appliance files; Linux and Darwin |

`direct` and `qemu` share all domain logic through [`../core`](../core/), which
runs standard util-linux/LVM/cryptsetup tools over a [`../runtime`](../runtime/)
transport — host-local for `direct`, an RPC to `kc-agent` for `qemu`. A plugin
adds only what genuinely differs: device attachment (loop devices vs virtio-blk)
and process lifecycle. `guestfs` wraps libguestfs directly and does not use core.

## Isolation rules

- Plugins must not import each other or share mutable package state.
- Linux-only plugins (`direct`, `guestfs`) call `backend.Factories.Register` only
  when `runtime.GOOS == "linux"`; Darwin lists `qemu` only.
- Host `qemu` must not import the linux-only appliance agent (`pkg/agent`);
  shared RPC types live in [`pkg/agent/protocol`](../../agent/protocol/).

## Adding a plugin

1. New package under `pkg/guest/plugins/<name>/` implementing `backend.Backend` +
   `backend.Factory` (reuse [`../core`](../core/) for domain logic where possible).
2. Optional: `backend.SharedSessionFactory`, `backend.ClevisAwareFactory`.
3. `init() { backend.Factories.Register("<name>", …) }` (guard on `runtime.GOOS`
   for platform-limited backends).
4. Blank-import from the stage binaries and `kc-v2v` mains.
