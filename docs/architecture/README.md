# Architecture Reference

Design and reference documentation for kc-utils. For contributor coding rules
(block isolation, backend transparency, plugin conventions), see
[community/architecture.md](../../community/architecture.md).

## Design docs

- [privilege-model.md](privilege-model.md) — host-mount vs libguestfs appliance privilege trade-offs
- [guest-os-handlers.md](guest-os-handlers.md) — Linux distro and Windows version classification, special cases, and code map
- [conversion-paths.md](conversion-paths.md) — OS + source-hypervisor path reference (index)
  - [conversion-paths-linux.md](conversion-paths-linux.md) — Linux cleanup and install matrices
  - [conversion-paths-windows.md](conversion-paths-windows.md) — Windows cleanup and install matrices
- [ref-baseline/README.md](ref-baseline/README.md) — MTV cold-migration ref vs kc-v2v baseline (plus dashboard)
- [../../tests/scenarios/virt-v2v-vs-kc-v2v.md](../../tests/scenarios/virt-v2v-vs-kc-v2v.md) — virt-v2v vs kc-v2v comparison narrative

## Pluggable Architecture

Blocks marked "pluggable" in the pipeline docs use Go interfaces registered
via a generic `Registry[K,V]`. Implementations live under
`pkg/<utility>/<block>/plugins/` and self-register in their `init()` functions.

All pipeline blocks live under `pkg/<utility>/<block>/`. The thin
`pkg/cmd/<utility>/pipeline.go` orchestrator wires them together.

Shared helpers live under `pkg/common/`. See [pkg/README.md](../../pkg/README.md).

Each binary links only the plugins it needs via blank imports in
`cmd/<binary>/main.go`.

## Code layout

| Layer | Index | Path pattern |
|-------|-------|--------------|
| Blocks | [pkg/README.md](../../pkg/README.md) | `pkg/<utility>/<block>/` |
| Block plugins | `<block>/plugins/README.md` | `pkg/<utility>/<block>/plugins/<impl>/` |
| Orchestrators | [pkg/cmd/](../../pkg/cmd/) | `pkg/cmd/<utility>/pipeline.go` |
| Shared helpers | [pkg/common/README.md](../../pkg/common/README.md) | `pkg/common/<semantic>/` |
| V2V libraries | [pkg/v2v/README.md](../../pkg/v2v/README.md) | `pkg/v2v/<package>/` |

Each utility has one README at `pkg/<utility>/README.md`. Pluggable blocks document
implementers in `plugins/README.md`, not per-impl READMEs.

## App docs

CLI flags, pipeline block tables, and usage guides live under
[../apps/README.md](../apps/README.md).
