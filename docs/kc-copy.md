# kc-copy Utility

NFC disk copy stage for blank-PVC population in Forklift vSphere migrations.
`kc-v2v` invokes it as a subprocess (same pattern as `kc-prepare`); it is also
usable standalone for copy-only debugging. See [docs/README.md](README.md) for
the conversion flow and [kc-v2v.md](kc-v2v.md) for orchestration.

Runs the copy executor from [`pkg/copy/`](../pkg/copy/) via
`copy.Run(CopyInput)`.

Uses pure Go govmomi NFC export -- no VDDK, nbdkit, or nbdcopy required.

## Entry Point

`cmd/kc-copy/main.go` — minimal flags or `--input` JSON; calls `copy.Run()` directly.
Does **not** load full `V2V_*` config or decide whether copy is needed — that
stays in `kc-v2v` (`env.NeedsCopy`, `ValidateCopyMode`, `ResolveCopySources`,
`ValidateCopySourceCount`).

## CLI

`kc-copy` has no subcommands. Two input modes:

1. **`--input copy-input.json`** — read `CopyInput` directly
2. **Minimal flags** — `--libvirt-url`, `--vm-name`, `--fingerprint`, `--disk-path`

Exit codes:

- `0` — copy completed
- `1` — configuration or copy error

## Usage

```bash
make build-kc-copy

export V2V_libvirtURL=vpx://user@vcenter/...
export V2V_vmName=my-vm
export V2V_fingerprint=...

kc-copy \
  --libvirt-url "$V2V_libvirtURL" \
  --vm-name "$V2V_vmName" \
  --fingerprint "$V2V_fingerprint" \
  --disk-path "[ds] vm/disk1.vmdk,[ds] vm/disk2.vmdk"
```

Or with JSON input:

```bash
cat > copy-input.json <<'EOF'
{
  "vcenter_url": "vpx://user@vcenter/...",
  "vm_name": "my-vm",
  "fingerprint": "...",
  "source_disks": ["[ds] vm/disk1.vmdk"],
  "workdir": "/var/tmp/v2v"
}
EOF

kc-copy --input copy-input.json
```

`source_disks` / `--disk-path` selects which NFC lease disks to copy (required).

The orchestrator (`kc-v2v`) decides whether copy is needed via `env.NeedsCopy()`
(`!V2V_inPlace`; default copy), validates disk state with
`env.ValidateCopyMode()`, resolves source disks via `env.ResolveCopySources()`,
gates source count vs empty PVCs with `env.ValidateCopySourceCount()`,
writes `copy-input.json`, and runs `kc-copy --input …` from `KC_BIN_DIR`
(default `/usr/lib/kc-utils`). See [kc-v2v.md](kc-v2v.md).

## Configuration

| Flag / JSON field | Env fallback | Default |
|-------------------|--------------|---------|
| `--libvirt-url` / `vcenter_url` | `V2V_libvirtURL` | (required) |
| `--vm-name` / `vm_name` | `V2V_vmName` | (required) |
| `--fingerprint` / `fingerprint` | `V2V_fingerprint` | (required) |
| `--disk-path` / `source_disks` | `V2V_diskPath` | (required) — VMDKs to select from the NFC lease |
| `--work-dir` / `workdir` | — | `/var/tmp/v2v` |
| `--copy-concurrency` / `copy_concurrency` | `V2V_copyConcurrency` | `4` (capped at disk count; `1` = sequential) |

vSphere credentials (Forklift conversion pod layout):

| Path | Purpose |
|------|---------|
| `/etc/secret/accessKeyId` | vSphere username |
| `/etc/secret/secretKey` | vSphere password |

Progress is written to `$WORKDIR/copy-progress.json` (default workdir:
`/var/tmp/v2v`).

## Copy Flow

For each selected source disk → empty PVC target (up to `copy_concurrency` disks in parallel):

1. Connect to vCenter via govmomi, export VM via NFC lease
2. Filter lease URLs to disks matching `source_disks` (normalized path; list order preserved)
3. Require selected count equals empty target count
4. Per disk (worker pool): HTTP GET NFC URL → `StreamToRaw` VMDK-to-raw converter → direct write to target
5. On first disk failure, cancel remaining in-flight copies
6. For block device targets, `fsync` the device before closing
7. Complete NFC lease

Count gates:

| Stage | Gate |
|-------|------|
| `kc-v2v` | `len(ResolveCopySources)` == empty PVC count |
| `kc-copy` | `len(FilterDiskURLs(...))` == empty PVC count |

### Disk selection and PVC ordering

1. `source_disks` lists the VMDKs to copy (from inventory / `V2V_diskPath`).
2. `mapDiskURLs` labels each NFC disk URL with the VM backing `FileName` (normalized)
   and drops non-disk lease items (nvram, etc. via `HttpNfcLeaseDeviceUrl.disk`).
3. NFC lease items are matched to `source_disks` by normalized path
   (`disk-000001.vmdk` → `disk.vmdk`).
4. Selected disks keep **source_disks order**; lease disks not in the list are skipped.
5. Empty PVC targets are sorted by numeric index (`/dev/blockN` or `/mnt/disks/diskN`).
6. Pairing is `targets[i]` ← `selected[i]`.

Forklift must attach PVCs in the same order as `source_disks` (same contract as
virt-v2v). Omitting a disk from the list (e.g. a shared disk) skips copying it.

### VMDK stream-to-raw conversion

`StreamToRaw` (`pkg/copy/vmdk.go`) parses the stream-optimized VMDK returned by
the NFC export and writes decompressed raw disk data directly to the target
(block device or file). The stream contains three element types:

| Element | Marker format | Alignment |
|---------|---------------|-----------|
| Compressed grain | 12-byte header (LBA + size) + zlib data | marker + data padded to sector boundary |
| Metadata (grain table / grain directory / footer) | Full 512-byte sector (numSectors + 0 + type + padding) + numSectors × 512 bytes payload | sector-aligned marker and payload |
| EOS | 12-byte header (0 + 0) + 4-byte type (0) | — |

Zero regions are left sparse (the block device or file already contains zeros).

Target paths (conversion pod `podVolumeMounts`):

| PVC volumeMode | Write path |
|----------------|------------|
| Block | `/dev/block{N}` |
| Filesystem | `/mnt/disks/disk{N}/disk.img` |

## Image / local layout

| Path | Role |
|------|------|
| `/usr/lib/kc-utils/kc-copy` | Pipeline stage binary (`kc-v2v` `BinDir`) |
| `/usr/bin/kc-copy` | Convenience path for manual debug |

Locally, `make build` places `kc-copy` in `bin/` next to the other pipeline
binaries. Point `kc-v2v` at that directory with `KC_BIN_DIR=$PWD/bin`.

## Related

- [kc-v2v.md](kc-v2v.md) — full orchestrator including copy gate
- [pkg/v2v/README.md](../pkg/v2v/README.md) — package internals
