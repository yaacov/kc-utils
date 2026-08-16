# Guest Disk Backends

kc-utils accesses guest filesystems through a pluggable **backend**. Every
stage (`kc-prepare`, `kc-convert-*`, `kc-finalize`) reaches guest disks only
through the `Guest` facade in [`pkg/guest/`](../../pkg/guest/README.md); the
facade delegates to the active backend, so the rest of the codebase is unaware
of which one is running. Backends differ in **how** they mount and operate on
guest disks — and therefore in the host privileges, hardware, and image size
they require.

| Backend | Mechanism | Requires | Page |
|---------|-----------|----------|------|
| **direct** | Host-kernel mounts (`losetup`, LVM, `cryptsetup`, `mount`) | `CAP_SYS_ADMIN` / privileged pod; Linux only | [direct.md](direct.md) |
| **guestfs** | libguestfs appliance; guest FS via `guestfish` RPC | `/dev/kvm`, unprivileged pod; Linux only | [guestfs.md](guestfs.md) |
| **qemu** | QEMU appliance + `kc-agent` RPC | QEMU + appliance artifacts; Linux and Darwin | [qemu.md](qemu.md) |

Both appliance backends run the same conversion tools (mount, LVM, cryptsetup,
fsck, hivex) inside a throwaway VM instead of on the host. The two appliances
differ: **guestfs** uses the [libguestfs/supermin appliance](guestfs.md);
**qemu** uses the kc-utils [QEMU appliance](appliance.md) driven by
[`kc-agent`](kc-agent.md).

## Selecting a backend

Selection is **required** — there is no default:

```bash
--backend <name>      # kc-prepare / kc-convert-* / kc-finalize
V2V_backend=<name>    # kc-v2v orchestrator
```

The runtime list comes from the backends registered in the binary
(`guest.Factories.List()`). `direct` and `guestfs` register only when
`runtime.GOOS == "linux"`; `qemu` registers on Linux and Darwin. The `kc-v2v`
container image sets `V2V_backend=guestfs` explicitly.

## Comparison

| | direct (host-mount) | guestfs (libguestfs) | qemu (QEMU appliance) |
|-|---------------------|----------------------|-----------------------|
| **Host privileges** | `CAP_SYS_ADMIN` / privileged | `/dev/kvm` only | `/dev/kvm` on Linux (else TCG); no `CAP_SYS_ADMIN` |
| **Performance** | Near-native (direct kernel I/O) | Appliance boot + RPC latency | Appliance boot + RPC latency |
| **KVM/accel** | Not used | Required (KVM) | `/dev/kvm` on Linux, `hvf` on macOS, `tcg` only when hardware acceleration is unavailable |
| **Image size** | Smallest (no QEMU/kernel) | Larger (QEMU + supermin appliance) | Larger (QEMU + kc-appliance artifacts) |
| **Isolation** | Weak (guest FS drivers in host kernel) | Strong (VM boundary) | Strong (VM boundary) |
| **Platforms** | Linux | Linux | Linux, macOS |

For the full privilege breakdown (which operations need which capabilities), see
[direct.md](direct.md#privileged-capabilities).

## Cross-cutting topics

- [appliance.md](appliance.md) — the QEMU appliance artifacts and how they are built
- [kc-agent.md](kc-agent.md) — the in-appliance agent and its RPC protocol
- [clevis-nbde.md](clevis-nbde.md) — Clevis/NBDE LUKS unlock per backend
- [../architecture/filesystem-checks.md](../architecture/filesystem-checks.md) —
  per-backend `FSCheck` command matrix and check-vs-repair semantics

## For contributors

Backend transparency, isolation rules, and how to add a backend are code-level
concerns documented in
[community/architecture.md](../../community/architecture.md#backend-transparency)
and [pkg/guest/README.md](../../pkg/guest/README.md).
