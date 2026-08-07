# kc-utils: Conversion Flow

## Pipeline Overview

```
                  PrepareInput JSON
                          |
                          v
                  +---------------+
                  |  kc-prepare   |
                  +---------------+
                          |
                  PrepareOutput JSON
                  + mounted guest FS
                          |
             +------------+------------+
             |                         |
             v                         v
  +-------------------+    +-----------------------+
  | kc-convert-linux  |    | kc-convert-windows    |
  +-------------------+    +-----------------------+
             |                         |
      ConverterOutput JSON      ConverterOutput JSON
             |                         |
             +------------+------------+
                          |
                          v
                  +---------------+
                  |  kc-finalize  |
                  +---------------+
                          |
                  TargetMeta JSON
```

The orchestrator (Forklift, `kc-v2v`, a shell script, or any external caller)
runs the four binaries in sequence. After `kc-prepare` completes, the orchestrator
reads `PrepareOutput.converter` to decide whether to invoke `kc-convert-linux`
or `kc-convert-windows`.

## V2V orchestration

For Forklift migrations, [kc-v2v](kc-v2v.md) is the conversion-pod entrypoint.
It runs the optional copy stage when needed, then the same four-binary pipeline:

```text
Conversion pod
  → kc-v2v (pkg/v2v: config, env, vsphere, inspection)
      → env.NeedsCopy?  → ResolveCopySources → ValidateCopySourceCount → write copy-input.json → kc-copy
      → DiscoverDisks
      → kc-prepare → kc-convert-* → kc-finalize
      → inspection XML + HTTP :8080
```

Set `virt_v2v_image_fqin` to the kc-v2v container image. `kc-v2v` loads `V2V_*`
env vars, discovers disks on conversion-pod PVC mounts, and spawns the pipeline
binaries (including `kc-copy` when copy is needed). See
[forklift-usage.md](forklift-usage.md) for full MTV / Forklift setup.

To run copy alone (without conversion), use [kc-copy.md](kc-copy.md).

## Quick start

Runnable examples and sample JSON for every handoff live in
[examples/](examples/README.md).

From a Linux host with root, after `make build`
(see [privilege-model.md](privilege-model.md) for why root is required and how
libguestfs avoids it):

```bash
sudo docs/examples/run-linux-disk.sh
```

That creates a test disk, runs all four binaries, and prints output JSON paths.
See [examples/README.md](examples/README.md) for individual input/output samples,
multiboot recovery, and Windows virtio-win setup.

## Data Types

| Type | Produced By | Consumed By | Contents |
|------|-------------|-------------|----------|
| `PrepareInput` | Orchestrator | kc-prepare | Disk paths, source metadata, LUKS keys, `options.root`, other options |
| `PrepareOutput` | kc-prepare | converter + kc-finalize | OS info, firmware, disk layout, root device, mount paths, converter name |
| `ConverterOutput` | kc-convert-linux / kc-convert-windows | kc-finalize | GuestCaps (virtio, display, etc.) |
| `TargetMeta` | kc-finalize | Orchestrator | Everything needed to create the target KVM VM |

## CLI Invocation

A typical conversion sequence:

```bash
# 1. Prepare: open disks, inspect OS, mount filesystems
kc-prepare \
  --input      /tmp/kc/prepare-input.json \
  --output     /tmp/kc/prepare-output.json \
  --mount-root /mnt/guest

# 2. Convert (orchestrator picks based on PrepareOutput.converter)
kc-convert-linux \
  --prepare-data /tmp/kc/prepare-output.json \
  --output       /tmp/kc/converter-output.json \
  --mount-root   /mnt/guest

# or for Windows guests:
kc-convert-windows \
  --prepare-data /tmp/kc/prepare-output.json \
  --output       /tmp/kc/converter-output.json \
  --mount-root   /mnt/guest

# 3. Finalize: unmount, trim, fsck, assign buses, emit metadata
kc-finalize \
  --prepare-data /tmp/kc/prepare-output.json \
  --convert-data /tmp/kc/converter-output.json \
  --output       /tmp/kc/target-meta.json \
  --mount-root   /mnt/guest
```

All binaries also accept `--log-level` (`debug`, `info`, `warn`, `error`).
The converter binaries accept `--offline` to skip network-dependent steps.

Sample JSON files: [examples/](examples/README.md).

## Communication Pattern

### JSON Files

Each binary reads its inputs from JSON files on disk and writes its outputs to a
new JSON file. The orchestrator passes file paths via CLI flags. This keeps
each binary stateless and independently testable.

```
/tmp/kc/
  prepare-input.json      written by orchestrator
  prepare-output.json     written by kc-prepare
  converter-output.json   written by kc-convert-linux or kc-convert-windows
  target-meta.json        written by kc-finalize
```

### Shared Mount Directory

The guest root filesystem is mounted at a host path (e.g., `/mnt/guest`)
by `kc-prepare`. Subsequent binaries access guest files through this mount point.
`kc-finalize` is responsible for unmounting everything in reverse order.

### Multiboot and root selection

When a guest has more than one operating system, `kc-prepare` defaults to
picking the first discovered root (`options.root` omitted or `"first"`). Set
`options.root` to `"single"` to fail with `root_candidates` listed, or to a
device path (for example `"/dev/loop0p2"`) to choose explicitly. Only the
chosen OS filesystem tree is mounted under `--mount-root`.

Mount ordering uses path-length sorting so that nested mounts (e.g., `/boot`
before `/boot/efi`) are mounted in the correct order and unmounted in reverse.

## Who Runs What

### V2V path (Forklift / kc-v2v)

| Step | Actor | Action |
|------|-------|--------|
| 0 | kc-v2v | If `env.NeedsCopy()` (`!V2V_inPlace`, default): resolve sources, validate count vs empty PVCs, write `copy-input.json`, spawn `kc-copy` |
| 1 | kc-v2v | Discover disks on PVC mounts, write `PrepareInput` JSON, invoke `kc-prepare` |
| 2 | kc-v2v | Read `PrepareOutput.converter`, invoke `kc-convert-linux` or `kc-convert-windows` |
| 3 | kc-v2v | Invoke `kc-finalize` |
| 4 | kc-v2v | Write inspection XML; serve HTTP when `LOCAL_MIGRATION=true` |
| 5 | Forklift | Read inspection / TargetMeta, create the target VM |

### Direct path (pre-filled disks)

| Step | Actor | Action |
|------|-------|--------|
| 1 | Orchestrator | Write `PrepareInput` JSON, invoke `kc-prepare` |
| 2 | Orchestrator | Read `PrepareOutput.converter` field |
| 3 | Orchestrator | Invoke `kc-convert-linux` or `kc-convert-windows` |
| 4 | Orchestrator | Invoke `kc-finalize` |
| 5 | Orchestrator | Read `TargetMeta` JSON, create the target VM |

## Pluggable Architecture

Blocks marked "pluggable" in the pipeline docs use Go interfaces registered
via a generic `Registry[K,V]`. Implementations live under
`pkg/<utility>/<block>/plugins/` and self-register in their `init()` functions.

All pipeline blocks live under `pkg/<utility>/<block>/`. The thin
`pkg/cmd/<utility>/pipeline.go` orchestrator wires them together.

Shared helpers live under `pkg/common/`. See [pkg/README.md](../pkg/README.md).

Each binary links only the plugins it needs via blank imports in
`cmd/<binary>/main.go`.

## Detailed Pipeline Documentation

### Conversion pipeline

- [kc-prepare.md](kc-prepare.md) — 9 blocks
- [kc-convert-linux.md](kc-convert-linux.md) — 11 blocks
- [kc-convert-windows.md](kc-convert-windows.md) — 9 blocks
- [kc-finalize.md](kc-finalize.md) — 6 blocks

### V2V orchestration

- [kc-v2v.md](kc-v2v.md) — Forklift conversion pod entrypoint (copy + pipeline + HTTP)
- [kc-copy.md](kc-copy.md) — NFC disk copy stage CLI (subprocess + standalone)
- [forklift-usage.md](forklift-usage.md) — Using kc-v2v with Forklift (MTV)

### Design

- [guest-os-handlers.md](guest-os-handlers.md) — Linux distro and Windows version classification, special cases, and code map
- [privilege-model.md](privilege-model.md) — host-mount vs. libguestfs appliance privilege trade-offs
- [ref-baseline/README.md](ref-baseline/README.md) — MTV cold-migration ref vs kc-v2v baseline (plus dashboard)

## Code layout

| Layer | Index | Path pattern |
|-------|-------|--------------|
| Blocks | [pkg/README.md](../pkg/README.md) | `pkg/<utility>/<block>/` |
| Block plugins | `<block>/plugins/README.md` | `pkg/<utility>/<block>/plugins/<impl>/` |
| Orchestrators | [pkg/cmd/](../pkg/cmd/) | `pkg/cmd/<utility>/pipeline.go` |
| Shared helpers | [pkg/common/README.md](../pkg/common/README.md) | `pkg/common/<semantic>/` |
| V2V libraries | [pkg/v2v/README.md](../pkg/v2v/README.md) | `pkg/v2v/<package>/` |

Each utility has one README at `pkg/<utility>/README.md`. Pluggable blocks document
implementers in `plugins/README.md`, not per-impl READMEs.
