# kc-utils Architecture Guide

Canonical architecture guide for contributors and agents. Read this before modifying code structure, splitting/merging files, adding packages, or refactoring.

## Structure: Stages and Blocks

kc-utils is built from **stages** and **blocks**:

- **Stage** — a pipeline step shipped as its own binary (`kc-prepare`, `kc-convert-linux`, `kc-finalize`, …). The stage orchestrator wires blocks together and handles I/O between pipeline steps (JSON on disk, shared mount).
- **Block** — an independent unit of logic inside a stage (hypervisor removal, driver install, distro detection, …). Block code lives in `pkg/<stage>/`; the stage orchestrator lives in `pkg/cmd/<stage>/pipeline.go`.

```
kc-prepare → kc-convert-linux / kc-convert-windows → kc-finalize
  (stage)              (stage)                              (stage)
     │                     │                                    │
  blocks                blocks                               blocks
(pkg/prepare/)   (pkg/convert-linux|convert-windows/)    (pkg/finalize/)
```

## Block Isolation

Each block is a self-contained code unit with a typed input/output **contract**. An agent (human or AI) working on one block should only need that contract — not the internals of sibling blocks.

This means:
- A block receives typed inputs and produces typed outputs. That contract is the only coupling between blocks.
- Blocks do not import or share implementation code with other blocks.
- Never reach into another block's internal state, files, or global variables.
- Blocks must have **no side effects** that leak into other blocks.

### Sharing code

Prefer **not** sharing implementation — readability and maintainability come first; no early optimization. When two blocks look similar, duplicating a small amount of code is often the right call.

**Types and structs are the exception.** Shared types are the contracts between blocks and stages; they belong in `pkg/common/` (e.g. `pkg/common/types/`).

When implementation must be shared:

| Scope | Where it lives |
|-------|----------------|
| Between blocks in the same stage | Stage level — the orchestrator (`pkg/cmd/<stage>/`) or stage-local helpers under `pkg/<stage>/`, not imports between block packages |
| Between stages | `pkg/common/` — types, registries, and other cross-stage infrastructure |
| Guest disk access (all stages) | `pkg/guest/` — the `Backend` abstraction; blocks use the `Guest` handle, never the backend directly |

If block B needs output from block A, the stage orchestrator passes it explicitly — block B does not import block A.

## Pipeline Stage Isolation

The pipeline runs four **stage binaries** as OS subprocesses (see diagram above).

Each stage binary has its own:
- `cmd/kc-*/main.go` — CLI entry point with blank imports for plugin loading
- `pkg/cmd/<stage>/pipeline.go` — stage orchestrator; wires blocks together
- `pkg/<stage>/` — block packages
- `docs/apps/kc-<stage>.md` — CLI flags, block descriptions, and behavioral notes

**Hard rule:** No `pkg/<stage>/` package may import from another stage's `pkg/<other-stage>/`. Stages communicate exclusively through JSON files on disk and a shared mount point. This makes each binary independently compilable, testable, and deployable.

Cross-stage shared code lives in:
- `pkg/common/` — types (contracts), logger, plugin registry, config editors, compression, Windows registry access
- `pkg/guest/` — guest disk access abstraction (the Backend interface)
- `pkg/v2v/` — orchestrator libraries (config, env, inspection) used only by `kc-v2v`

## Guest disk backend plugins

`pkg/guest/` provides a `Backend` interface and a `Factories` registry. Built-in
implementations self-register via `init()`:

| Name | Implementation | Requires |
|------|----------------|----------|
| **direct** (default) | `pkg/guest/direct/` — host kernel mounts via `mount(8)`, `losetup`, LVM, cryptsetup | root / `CAP_SYS_ADMIN` |
| **guestfs** (`--backend=guestfs`) | `pkg/guest/guestfs/` — shared `guestfish --listen` session; guest FS via appliance RPC | `/dev/kvm` only |

CLI/env: `--backend <name>` / `V2V_backend`.
Available names come from `guest.Factories.List()` at runtime after blank imports.

Shared host helpers (not domain logic) live in `pkg/guest/common`. Backend packages
must not import each other or share mutable package state. Parent `pkg/guest`
must not import concrete backends — binaries blank-import them.

### Backend Transparency

The rest of the codebase must be **completely unaware** of which backend is active. All guest disk operations go through the `Guest` handle (`pkg/guest/guest.go`), which delegates to the active `Backend`.

Rules:
- **Never** call `guestfish`, `mount`, `umount`, `losetup`, `lvm`, `cryptsetup`, `chroot`, `fsck`, or `fstrim` for guest disks outside `pkg/guest/`. Orchestrators may call `guest.StartSharedSession` / `StartSharedListener` only.
- Block packages use `guest.Active()` or receive a `*guest.Guest` handle — they never check which backend is running.
- If behavior must differ by mode, that difference lives inside the `Backend` implementations, not in the calling block.
- The `Guest` struct's methods (`ReadFile`, `WriteFile`, `Exists`, `Glob`, `RunCommand`, etc.) are the file-system API that blocks should use.

### Guest Operations Used by Converters

Converters never call `chroot` or other privileged tools directly. `kc-convert-linux` uses `guest.RunInGuest(guestRoot, cmd)` for commands like `dracut` or `grub-mkconfig`; the active backend decides how to execute:

- **Direct** — real `chroot(2)` into the mounted guest root (`pkg/guest/direct/`)
- **Guestfs** — `guestfish "sh"` inside the appliance VM (`pkg/guest/guestfs/`)

`kc-convert-windows` reads virtio-win drivers from the pre-extracted driver tree
at `/usr/share/virtio-win/drivers/by-os/` on the host filesystem (not guest-disk I/O).
Driver files are later copied into the guest via the `Guest` handle.

## Plugin System

Pluggable blocks use a generic `plugin.Registry[K, V]` with `init()` self-registration:

```
pkg/<stage>/<block>/
    <block>.go          — interface + registry variable
    plugins/
        <impl>/
            <impl>.go   — init() registers into the registry
```

To add a new hypervisor, distro, driver source, etc.: drop a file into `plugins/` with an `init()` function. The binary's `main.go` blank-imports it.

**Each stage owns its own plugin registries.** Even when two stages need the same interface (e.g., UEFI editor, firstboot handler), each stage may maintain its own registry. This is intentional — it keeps stage binaries independent. Shared interfaces can live in `pkg/common/` when both stages genuinely share the contract, but the default is stage-local ownership.

## Priorities When Modifying Code

In order of importance:

### 1. Block isolation and no side effects
Each block is a self-contained unit with a clear contract. An agent working on block A should only need to know: what types go in, what types come out, and what `Guest` methods are available. No shared implementation between blocks; no global state; no hidden channels.

### 2. Readability and maintainability
Code should be scannable. A developer landing in a pipeline file should immediately see the sequence of operations. Prefer:
- Named helper functions over long inline blocks
- Clear function names that describe *what* the step does
- Explicit data flow (pass values, don't rely on shared mutable state)

### 3. Code blocks and plugins over short code
Prefer the block/plugin structure even when it means more files or a few extra lines. A 15-line plugin file that self-registers is better than inlining the logic and saving 5 lines. The plugin structure makes the system extensible and keeps blocks independent.

**Do not** optimize for line count. Do not merge blocks to make code shorter if it creates coupling. Do not inline plugin logic to save a file. The file/package overhead is the cost of isolation — it is worth it.

### 4. Backend transparency
All disk access goes through `pkg/guest/`. Never leak backend-specific logic into blocks.

### 5. Stage isolation
No cross-stage imports under `pkg/`. Stages (binaries) talk via JSON files; blocks within a stage talk via explicit parameters and return values through the stage orchestrator.

### 6. Keep docs in sync

When modifying a pipeline block, update the corresponding `docs/apps/kc-<stage>.md` file (e.g., `docs/apps/kc-convert-linux.md`, `docs/apps/kc-prepare.md`). CLI flags, block descriptions, plugin tables, and behavioral notes must reflect the current code. A code change without a matching docs update is incomplete.

**Documentation layering** (see [docs/architecture/README.md](../docs/architecture/README.md#documentation-layering)) follows the same isolation as code:

- **Stage app docs** (`docs/apps/kc-*.md`) document only that binary's CLI and JSON contract. Put orchestrator mapping (`V2V_*` → handoff JSON), Forklift wiring, and "when spawned by kc-v2v" notes in an **Integration** section at the end—or in `docs/apps/kc-v2v.md`—not in the opening paragraph or primary config tables.
- **Orchestrator docs** (`kc-v2v.md`, `forklift-usage.md`, `build/kc-v2v/README.md`) own env translation, subprocess spawning, and deployment.
- **Architecture docs** (`docs/architecture/`) own cross-cutting design (privilege model, conversion paths, shared listener contract). Link from stage docs instead of duplicating kc-v2v lifecycle in every stage file.

Test: *If this binary were run standalone with only JSON input, would this sentence still belong here?*

## What NOT to Do

- **Don't share implementation prematurely.** Duplicate before extracting. Shared helpers within a stage belong at stage level; shared code across stages belongs in `pkg/common/`. Types are always shared — they are the contracts.
- **Don't inline trivial plugin files.** A plugin that is "just 10 lines" still serves the architectural purpose of being independently addable/removable via blank imports.
- **Don't add cross-block dependencies.** If block B needs something from block A, pass it through the pipeline orchestrator as an explicit parameter — don't import block A from block B.
- **Don't check which backend is active outside `pkg/guest/`.** No `if mode == ModeGuestfs` in block code.
- **Don't add side effects to blocks.** A block should not write global state, modify environment variables, or communicate with other blocks except through its return values.

## File Organization Conventions

- Path pattern: `<layer>/<stage>/<semantic-name>/` — stage matches the binary, semantic name describes what the block does.
- Stage orchestrators (`pkg/cmd/<stage>/pipeline.go`) wire blocks together, handle errors, and pass data. Logic lives in block packages under `pkg/<stage>/`.
- Test files live alongside the code they test (`*_test.go`).
- Each pluggable block has a `plugins/README.md` documenting available implementations.
