# kc-finalize Pipeline

Unmounts guest filesystems, trims free space, runs post-conversion filesystem
checks, assigns target bus slots, resolves firmware type, and assembles the
final `TargetMeta` JSON that contains everything the orchestrator needs to
create the KVM virtual machine.

Requires Linux (`//go:build linux`).

## Entry Point

`cmd/kc-finalize/main.go` — orchestrator in `pkg/cmd/finalize/pipeline.go`.

## CLI Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--input` | yes\* | | Path to PipelineData JSON (from kc-convert-linux or kc-convert-windows; contains `prepare` and `convert` sections) |
| `--output` | no | `target-meta.json` | Path to write PipelineData JSON (with `target` section added) |
| `--mount-root` | no | `/tmp/kc-guest` | Guest mount root (live mounts in direct mode; path key for `pkg/guest` in guestfs mode) |
| `--guestfs` | no | `false` | Use libguestfs appliance instead of privileged mounts (adopts shared `GUESTFISH_PID` from `kc-v2v`; does not exit it) |
| `--teardown-only` | no | `false` | Reclaim orphaned guest resources only (no metadata) |
| `--log-level` | no | `info` | Log level (`debug`, `info`, `warn`, `error`) |

\* Not required with `--teardown-only`. `--input` is optional then (mount-root-only fallback).

## Pipeline Blocks

Order matches [`pkg/cmd/finalize/pipeline.go`](../pkg/cmd/finalize/pipeline.go):

| # | Block | Type | Package | Description |
|---|-------|------|---------|-------------|
| 1 | Customize | pluggable: `Customizer` | `pkg/finalize/customize/` | Run post-mount customization (native Go or dynamic scripts) |
| 2 | Fstrim | inline | `pkg/guest/` | Trim mounted guest filesystems via `Guest.FSTrim()` |
| 3 | Teardown | inline | `pkg/guest/` | Unmount filesystems, deactivate LVM, close LUKS via `Guest.Teardown()` |
| 4 | Post-Fsck | inline | `pkg/guest/` | Post-conversion filesystem check via `Guest.FSCheck()` |
| 5 | Target | strict | `pkg/finalize/target/` | Resolve firmware type and assign disk/NIC bus slots |
| 6 | Metadata | strict | `pkg/finalize/metadata/` | Assemble and write TargetMeta JSON |

Note: Blocks 2-4 are implemented as inline calls to `pkg/guest.Guest` methods in the pipeline, not as separate packages.

## Input

- `PipelineData` JSON: contains `prepare` section (OS info, disk layout, mount paths, source metadata) and `convert` section (GuestCaps, converter-specific warnings)
- Guest access at `--mount-root` (Customize and Fstrim via `pkg/guest`; then teardown)

Example inputs: [examples/prepare-output-complete.json](examples/prepare-output-complete.json),
[examples/convert-output-linux.json](examples/convert-output-linux.json).

## Output

`TargetMeta` JSON containing everything needed to define the target VM:

- `guest_caps` -- virtio capabilities (block bus, net bus, feature flags)
- `target_firmware` -- resolved firmware type (BIOS or UEFI)
- `target_buses` -- disk slot assignments (virtio-blk, SCSI, or IDE)
- `target_nics` -- network interface configuration
- `disks` -- source disk layout
- `inspect` -- guest OS inspection results
- `warnings` -- non-fatal issues collected during conversion

Example: [examples/target-meta.json](examples/target-meta.json).

## Plugin Implementations

| Interface | Implementations |
|-----------|----------------|
| `Customizer` | `native` (hostname, timezone, SELinux relabel), `dynamicscripts` (user-provided scripts for Linux and Windows) |
| `FirstBootHandler` | `systemd` (implementation in `pkg/common/firstboot/`, used by `dynamicscripts` customizer) |

## Cleanup Sequence

kc-finalize must clean up resources in the correct order:

1. Run customization while filesystems are still mounted (guestfs: appliance RPC)
2. Call `Guest.Sync()` — no-op in both backends (direct mounts and guestfs writes are already live)
3. Trim mounted guest filesystems
4. Teardown: unmount filesystems (deepest first), deactivate LVM, close LUKS / clear mount root
5. Fsck filesystems on block devices
6. Assign bus slots and resolve firmware (metadata-only)
7. Write final TargetMeta JSON

### Failure cleanup (`--teardown-only`)

Used by `kc-v2v` when prepare/convert/finalize fails before a normal teardown.
This path reclaims **orphaned host resources only** — no Sync, customize, trim,
or TargetMeta — via `Guest.TeardownDiscard()` (required when qcow2 overlays will
be discarded).

If `--input` is missing or unusable, cleanup falls back to mount-root-only
reclamation.
