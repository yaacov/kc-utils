# kc-utils Documentation

Documentation for the kc-utils conversion pipeline, V2V orchestration, and
design reference.

## Apps

CLI reference, pipeline block tables, usage guides, and runnable examples:

- [apps/README.md](apps/README.md) — conversion flow overview
- [apps/kc-prepare.md](apps/kc-prepare.md), [apps/kc-convert-linux.md](apps/kc-convert-linux.md), [apps/kc-convert-windows.md](apps/kc-convert-windows.md), [apps/kc-finalize.md](apps/kc-finalize.md) — pipeline binaries
- [apps/kc-v2v.md](apps/kc-v2v.md), [apps/kc-copy.md](apps/kc-copy.md), [apps/forklift-usage.md](apps/forklift-usage.md) — V2V orchestration and Forklift integration
- [apps/kc-agent-sh.md](apps/kc-agent-sh.md) — interactive debug shell into a running QEMU appliance
- [apps/examples/](apps/examples/README.md) — JSON samples and runnable example
- [apps/macos-local.md](apps/macos-local.md) — copy a vSphere Linux VM on a Mac, convert with `--backend qemu`, boot with `qemu-system-x86_64`

## Backends

Guest disk backends (how each stage reaches guest filesystems), the QEMU
appliance, and the in-appliance agent:

- [backends/README.md](backends/README.md) — backend concept, selection, and comparison
- [backends/direct.md](backends/direct.md), [backends/guestfs.md](backends/guestfs.md), [backends/qemu.md](backends/qemu.md) — the three backends
- [backends/appliance.md](backends/appliance.md), [backends/kc-agent.md](backends/kc-agent.md) — QEMU appliance artifacts and `kc-agent`
- [backends/clevis-nbde.md](backends/clevis-nbde.md) — Clevis/NBDE LUKS unlock per backend

## Architecture

Design reference, OS classification, conversion-path matrices, and benchmarks:

- [architecture/README.md](architecture/README.md) — architecture docs index and **documentation layering**
- [architecture/guest-os-handlers.md](architecture/guest-os-handlers.md) — Linux distro and Windows version handlers
- [architecture/conversion-paths.md](architecture/conversion-paths.md) — OS + hypervisor path reference
- [architecture/ref-baseline/README.md](architecture/ref-baseline/README.md) — MTV cold-migration benchmark
- [Dashboard](https://htmlpreview.github.io/?https://github.com/yaacov/kc-utils/blob/main/docs/architecture/ref-baseline/dashboard.html) ([source](architecture/ref-baseline/dashboard.html))
