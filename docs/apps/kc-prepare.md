# kc-prepare Pipeline

Opens source disks, inspects the guest OS, mounts guest filesystems, and
produces metadata for downstream converters.

Requires Linux (`//go:build linux`).

## Entry Point

`cmd/kc-prepare/main.go` — orchestrator in `pkg/cmd/prepare/pipeline.go`.

## CLI Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--input` | yes | | Path to PrepareInput JSON |
| `--output` | no | `prepare-out.json` | Path to write PrepareOutput JSON |
| `--mount-root` | no | `/tmp/kc-guest` | Host directory for guest filesystem mounts |
| `--guestfs` | no | `false` | Use libguestfs appliance instead of privileged mounts |
| `--log-level` | no | `info` | Log level (`debug`, `info`, `warn`, `error`) |

In guestfs mode, prepare adopts the shared `guestfish --listen` session when
`GUESTFISH_PID` / `KC_GUESTFISH_PID` is set by `kc-v2v` (and fails if that PID
is dead). Standalone runs without a PID env start a process-local listener.
Prepare never exits a shared listener — `kc-v2v` closes it after finalize.
Guest filesystems are mounted inside the appliance; `--mount-root` is a path
key for `pkg/guest` helpers, not a populated host tree. File I/O uses guestfish
RPC against those appliance mounts.

Root selection is configured in `PrepareInput.options.root` (not a CLI flag).

## Pipeline Blocks

Order matches [`pkg/cmd/prepare/pipeline.go`](../../pkg/cmd/prepare/pipeline.go):

| # | Block | Type | Package | Description |
|---|-------|------|---------|-------------|
| 1 | Validate | strict | `pkg/prepare/validate/` | Validate disks and create mount root |
| 2 | Guest | strict | `pkg/prepare/guest/` | Open disks, scan partitions, activate LVM (`luks/`, `overlay/`, `resolve/` subdirs) |
| 3 | Decrypt | inline | `pkg/guest/` | Open encrypted partitions via `Guest.Decrypt()` and `Guest.UnlockClevis()` |
| 4 | Pre-Fsck | inline | `pkg/guest/` | Pre-conversion filesystem check/repair via `Guest.FSCheck()` on unmounted partitions |
| 5 | Firmware | pluggable: `FirmwareDetector` | `pkg/prepare/firmware/` | Determine BIOS vs UEFI (also refreshed after mount) |
| 6 | Root | strict + pluggable selector | `pkg/prepare/root/` | Discover OS roots and apply `options.root` policy |
| 7 | Mount | strict + pluggable planner | `pkg/prepare/mount/` | Plan and execute guest filesystem mounts |
| 8 | Inspect | strict | `pkg/prepare/inspect/` | OS inspection (`/etc/os-release`, `/usr/lib/os-release`, redhat-release, debian_version), boot device, free space |
| 9 | Converter | pluggable: `ConverterSelector` | `pkg/prepare/converter/` | Choose converter binary |

## Filesystem checks

After LUKS decrypt and before root discovery or mount, prepare runs
`Guest.FSCheck()` on every partition on every disk while devices are still
unmounted. Supported filesystem types, check-vs-repair behavior per backend,
and failure handling are documented in
[../architecture/filesystem-checks.md](../architecture/filesystem-checks.md).
Fsck failures are non-fatal: prepare logs a warning and continues.

## Input

`PrepareInput` JSON containing:

- `disks` -- list of source disk paths or URIs
- `source` -- source hypervisor metadata (name, type, firmware hint, NICs, etc.)
- `luks` -- optional LUKS decryption keys
- `options.root` -- root selection policy (see below)
- `options.static_ips` -- static IPs to preserve (Windows)

### Root selection (`options.root`)

| Value | Behavior |
|-------|----------|
| *(omitted)* or `first` | Pick the first discovered root (default; Forklift / virt-v2v `--root first` compatible) |
| `single` | Fail if multiple OS roots are found; error lists candidates |
| `/dev/...` | Pick the root on the given block device |

Invalid values (including `ask`) are rejected. On explicit `single` multiboot failure,
`PrepareOutput` may include `root_candidates` and `error` when the output file is written.

Guestfs remount (convert/finalize) enriches missing secondary mounts via
`inspect-os` / `inspect-get-mountpoints`. That enrichment matches the preferred
root (device at `/` from prepare `root_device`) against the full `inspect-os`
list; if no preferred root is present it defaults to the first inspect root.
A preferred root that is not listed by `inspect-os` is an error. When prepare
and inspect disagree on which device owns a guest path, prepare wins.

Example inputs: [examples/prepare-input-linux.json](examples/prepare-input-linux.json),
[examples/prepare-input-multiboot.json](examples/prepare-input-multiboot.json).
Example outputs: [examples/prepare-output-complete.json](examples/prepare-output-complete.json),
[examples/prepare-output-error-multiboot.json](examples/prepare-output-error-multiboot.json).

## Output

`PrepareOutput` JSON containing:

- `converter` -- which converter binary to run
- `inspect` -- detected OS family, distro, version, architecture
- `firmware` -- detected firmware type (BIOS or UEFI)
- `disks` -- disk layout with per-partition mount points
- `mount_root` -- host path where the guest root is mounted at `/`
- `root_device` -- block device path of the chosen OS root
- `root_candidates` -- discovered roots (populated on multiboot errors)
- `boot_device` -- partition containing the bootloader

## Plugin Implementations

| Interface | Implementations |
|-----------|----------------|
| `Decryptor` | `keyfile`, `clevis` |
| `RootSelector` | `first` (default), `single`, `device` |
| `MountPlanner` | `linux` (fstab), `windows` |
| `FirmwareDetector` | `gpt-esp` |
| `ConverterSelector` | `default` |
