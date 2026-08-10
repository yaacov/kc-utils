# Architecture Reference

Design and reference documentation for kc-utils. For contributor coding rules
(block isolation, backend transparency, plugin conventions), see
[community/architecture.md](../../community/architecture.md).

## Documentation layering

Docs follow the same containment rules as code: **each binary or package doc
describes only its own contract**; composition and deployment live elsewhere.

| Layer | Where | Documents | Does not document |
|-------|--------|-----------|-------------------|
| **Stage / utility** | [../apps/kc-*.md](../apps/) | CLI flags, input/output JSON, in-process behavior | How the orchestrator builds JSON, Forklift env vars, sibling binaries |
| **Orchestrator** | [../apps/kc-v2v.md](../apps/kc-v2v.md), [../apps/forklift-usage.md](../apps/forklift-usage.md) | `V2V_*` → JSON mapping, subprocess order, Forklift wiring | Block/plugin internals (link to architecture + stage docs) |
| **Integration** | [../apps/README.md](../apps/README.md), [../../build/kc-v2v/README.md](../../build/kc-v2v/README.md) | Full pod flow, image contents, MTV setup | Per-flag reference (link to stage docs) |
| **Design** | This directory | Cross-cutting behavior (privilege, OS paths, benchmarks) | CLI tables duplicated from app docs |

**Rules (mirror [block isolation](../../community/architecture.md#block-isolation)):**

1. **Contract first** — A stage doc leads with `PrepareInput`, `CopyInput`, etc.
   Orchestrator-specific names (`V2V_*`, Forklift, `BuildCopyInput`) belong in
   `kc-v2v.md` or an **Integration** section at the end of the stage doc, not
   in the opening or config tables.
2. **One direction of dependency** — Stage docs may link up to the orchestrator;
   orchestrator docs link down to stage contracts. Avoid teaching Forklift TLS
   inside `kc-copy.md` (orchestrator maps that into `copy-input.json`).
3. **Shared contracts in architecture** — Cross-stage mechanisms (guestfish PID
   env, `PipelineData` shape, plugin registries) are described once here or in
   [community/architecture.md](../../community/architecture.md), with stage docs
   linking to the shared section instead of repeating orchestrator lifecycle.
4. **Image / MTV is not stage logic** — Container paths (`/usr/share/virtio-win/…`),
   Forklift Plan fields, and benchmark numbers live in `build/kc-v2v/` or
   `forklift-usage.md`, not in `kc-convert-*.md` or `guest-os-handlers.md`
   except as a one-line link.

When adding or editing docs, ask: *would this paragraph still be true if the
binary were invoked by a shell script with only JSON files?* If not, move it to
orchestrator or integration docs.

## Design docs

- [privilege-model.md](privilege-model.md) — host-mount vs libguestfs appliance privilege trade-offs
- [filesystem-checks.md](filesystem-checks.md) — guest fsck timing, supported FS types, check vs repair, Windows NTFS
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
