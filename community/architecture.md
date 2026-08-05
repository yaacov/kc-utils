# kc-utils Architecture Guide

Canonical architecture guide for contributors and agents. Read this before modifying code structure, splitting/merging files, adding packages, or refactoring.

## Core Principle: Self-Contained Code Blocks

Code is organized so that **each block can be worked on in isolation**. An agent (human or AI) working on one block should never need to understand the internals of another block — only its input/output interface. Blocks must have **no side effects** that leak into other blocks.

This means:
- A block receives typed inputs and produces typed outputs. That contract is the only coupling.
- Never reach into another block's internal state, files, or global variables.
- If two blocks need shared behavior, it goes in `pkg/common/` or `pkg/guest/` — the designated shared layers.
- When in doubt, duplicate a small amount of code rather than create a hidden dependency between blocks.

## Pipeline Stage Isolation

The pipeline is four **separate binaries** executed as OS subprocesses:

```
kc-prepare → kc-convert-linux / kc-convert-windows → kc-finalize
```

Each binary has its own:
- `cmd/kc-*/main.go` — CLI entry point with blank imports for plugin loading
- `internal/<stage>/pipeline.go` — thin orchestrator that wires blocks together
- `pkg/<stage>/` — block packages (the actual logic)
- `docs/kc-<stage>.md` — CLI flags, block descriptions, and behavioral notes

**Hard rule:** No `pkg/<stage>/` package may import from another stage's `pkg/<other-stage>/`. Stages communicate exclusively through JSON files on disk and a shared mount point. This makes each binary independently compilable, testable, and deployable.

The only cross-stage shared code lives in:
- `pkg/common/` — types, logger, plugin registry, config editors, compression, Windows registry access
- `pkg/guest/` — guest disk access abstraction (the Backend interface)
- `pkg/v2v/` — orchestrator libraries (config, env, inspection) used only by `kc-v2v`

## The Two Backends: Direct vs Guestfs

`pkg/guest/` provides two `Backend` implementations:

| Mode | Implementation | Requires |
|------|---------------|----------|
| **Direct** (default) | `direct_backend.go` — host kernel mounts via `mount(8)`, `losetup`, LVM, cryptsetup | root / `CAP_SYS_ADMIN` |
| **Guestfs** (`--guestfs`) | `guestfs_backend.go` — shared `guestfish --listen` session; guest FS via appliance RPC (`Checkout` for host-path tools) | `/dev/kvm` only |

### Backend Transparency

The rest of the codebase must be **completely unaware** of which backend is active. All guest disk operations go through the `Guest` handle (`pkg/guest/guest.go`), which delegates to the active `Backend`.

Rules:
- **Never** call `guestfish`, `mount`, `umount`, `losetup`, `lvm`, `cryptsetup`, `chroot`, `fsck`, or `fstrim` for guest disks outside `pkg/guest/`. Orchestrators may call `guest.StartSharedListener` only.
- Block packages use `guest.Active()` or receive a `*guest.Guest` handle — they never check which backend is running.
- If behavior must differ by mode, that difference lives inside the `Backend` implementations, not in the calling block.
- The `Guest` struct's methods (`ReadFile`, `WriteFile`, `Exists`, `Glob`, `RunCommand`, etc.) are the file-system API that blocks should use.

### Guest Operations Used by Converters

Converters never call `chroot` or other privileged tools directly. `kc-convert-linux` uses `guest.RunInGuest(guestRoot, cmd)` for commands like `dracut` or `grub-mkconfig`; the active backend decides how to execute:

- **Direct** — real `chroot(2)` into the mounted guest root (`pkg/guest/direct/`)
- **Guestfs** — `guestfish "sh"` inside the appliance VM (`pkg/guest/guestfs/`)

`kc-convert-windows` reads virtio-win drivers from the pre-extracted RPM tree
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
Each block is a self-contained unit. An agent working on block A should only need to know: what types go in, what types come out, and what `Guest` methods are available. No global state, no cross-block coupling, no hidden channels.

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
No cross-stage imports under `pkg/`. Stages talk via JSON files.

### 6. Keep docs in sync
When modifying a pipeline block, update the corresponding `docs/<stage>.md` file (e.g., `docs/kc-convert-linux.md`, `docs/kc-prepare.md`). CLI flags, block descriptions, plugin tables, and behavioral notes must reflect the current code. A code change without a matching docs update is incomplete.

## What NOT to Do

- **Don't create shared packages to eliminate small duplications across stages.** If `prepare/` and `finalize/` both have a 12-line registry wrapper, that duplication is acceptable — it keeps each stage self-contained. Only consolidate into `pkg/common/` when the shared code is substantial and both stages genuinely need the same contract.
- **Don't inline trivial plugin files.** A plugin that is "just 10 lines" still serves the architectural purpose of being independently addable/removable via blank imports.
- **Don't add cross-block dependencies.** If block B needs something from block A, pass it through the pipeline orchestrator as an explicit parameter — don't import block A from block B.
- **Don't check which backend is active outside `pkg/guest/`.** No `if mode == ModeGuestfs` in block code.
- **Don't add side effects to blocks.** A block should not write global state, modify environment variables, or communicate with other blocks except through its return values.

## File Organization Conventions

- Path pattern: `<layer>/<utility>/<semantic-name>/` — utility matches the binary, semantic name describes what the block does.
- Pipeline orchestrators are thin: they wire blocks together, handle errors, and pass data. Logic lives in `pkg/`.
- Test files live alongside the code they test (`*_test.go`).
- Each pluggable block has a `plugins/README.md` documenting available implementations.
