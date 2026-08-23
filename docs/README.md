# kc-utils Documentation

Documentation for the kc-utils conversion pipeline, V2V orchestration, and
design reference.

## Apps

CLI reference, pipeline block tables, usage guides, and runnable examples:

- [apps/README.md](apps/README.md) — conversion flow overview
- [apps/kc-prepare.md](apps/kc-prepare.md), [apps/kc-convert-linux.md](apps/kc-convert-linux.md), [apps/kc-convert-windows.md](apps/kc-convert-windows.md), [apps/kc-finalize.md](apps/kc-finalize.md) — pipeline binaries
- [apps/kc-v2v.md](apps/kc-v2v.md), [apps/kc-copy.md](apps/kc-copy.md), [apps/forklift-usage.md](apps/forklift-usage.md) — V2V orchestration and Forklift integration
- [apps/kc-guest-agent.md](apps/kc-guest-agent.md) — in-appliance PID 1 for `--backend qemu`
- [apps/examples/](apps/examples/README.md) — JSON samples and runnable example

## Debug

Local Mac/Linux cookbook for `--backend qemu` (fetch VMware disks, hold the
appliance, prepare / convert / finalize, boot the converted x86 guest):

- [debug/README.md](debug/README.md) — prerequisites, held appliance, debug-socket attach

## Architecture

Design reference, OS classification, conversion-path matrices, and benchmarks:

- [architecture/README.md](architecture/README.md) — architecture docs index and **documentation layering**
- [architecture/backends.md](architecture/backends.md) — guest disk backends (`direct`, `guestfs`, `qemu`)
- [architecture/guest-os-handlers.md](architecture/guest-os-handlers.md) — Linux distro and Windows version handlers
- [architecture/conversion-paths.md](architecture/conversion-paths.md) — OS + hypervisor path reference
- [architecture/ref-baseline/README.md](architecture/ref-baseline/README.md) — MTV cold-migration benchmark
- [Dashboard](https://htmlpreview.github.io/?https://github.com/yaacov/kc-utils/blob/main/docs/architecture/ref-baseline/dashboard.html) ([source](architecture/ref-baseline/dashboard.html))
