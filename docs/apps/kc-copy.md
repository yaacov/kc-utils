# kc-copy Utility

Standalone vSphere NFC disk copy: downloads VMDK disks from vCenter/ESXi over
HTTPS and writes raw images to empty PVC targets (block devices or filesystem
images). Pure Go govmomi export — no VDDK, nbdkit, or nbdcopy.

Runs [`pkg/copy/`](../../pkg/copy/) via `copy.Run(CopyInput)`.

**Entry:** `cmd/kc-copy/main.go` — flags or `--input` JSON; no subcommands.

## CLI

Two input modes:

1. **`--input copy-input.json`** — read `CopyInput` directly
2. **Flags** — `--host`, `--vm-name`, `--fingerprint`, `--disk-path`, TLS flags, etc.

Exit codes: `0` success, `1` configuration or copy error.

## Usage

```bash
make build-kc-copy

kc-copy \
  --host vcenter.example.com \
  --datacenter mydatacenter \
  --insecure \
  --vm-name my-vm \
  --fingerprint "AB:CD:..." \
  --disk-path "[ds] vm/disk1.vmdk,[ds] vm/disk2.vmdk" \
  --output copy-progress.json
```

JSON input (skip TLS verification):

```bash
cat > copy-input.json <<'EOF'
{
  "host": "vcenter.example.com",
  "datacenter": "mydatacenter",
  "insecure": true,
  "vm_name": "my-vm",
  "fingerprint": "...",
  "source_disks": ["[ds] vm/disk1.vmdk"],
  "workdir": "/var/tmp/copy",
  "output_path": "copy-progress.json"
}
EOF

kc-copy --input copy-input.json
```

Secure TLS with a custom CA PEM file:

```bash
cat > copy-input.json <<'EOF'
{
  "host": "vcenter.example.com",
  "datacenter": "mydatacenter",
  "ca_cert": "/path/to/ca.pem",
  "vm_name": "my-vm",
  "fingerprint": "...",
  "source_disks": ["[ds] vm/disk1.vmdk"],
  "workdir": "/var/tmp/copy",
  "output_path": "copy-progress.json"
}
EOF

kc-copy --input copy-input.json
```

`source_disks` / `--disk-path` selects which NFC lease disks to copy (required).

## Configuration

| Flag / JSON field | Default | Description |
|-------------------|---------|-------------|
| `--host` / `host` | (required) | vCenter or ESXi hostname |
| `--datacenter` / `datacenter` | — | vSphere datacenter name (optional) |
| `--insecure` / `insecure` | `false` | Skip TLS certificate verification |
| `--ca-cert` / `ca_cert` | — | PEM custom CA path; error if set but file missing |
| `--vm-name` / `vm_name` | (required) | Source VM name |
| `--fingerprint` / `fingerprint` | (required) | vCenter SSL thumbprint fallback for SDK TLS |
| `--disk-path` / `source_disks` | (required) | VMDKs to select from the NFC lease |
| `--work-dir` / `workdir` | `/var/tmp/v2v` | Working directory |
| `--output` / `output_path` | `copy-progress.json` | Progress output file |
| `--copy-concurrency` / `copy_concurrency` | `4` | Max parallel disk copies (capped at disk count; `1` = sequential) |
| `--log-level` | `info` | Log level (`debug`, `info`, `warn`, `error`) |

vSphere credentials (read at connect time, not part of `CopyInput`):

| Path | Purpose |
|------|---------|
| `/etc/secret/accessKeyId` | vSphere username |
| `/etc/secret/secretKey` | vSphere password |

### TLS modes

TLS is resolved from **`insecure`** and **`ca_cert` only** (flags or JSON).
Omit both for system CA (Go host trust store).

| Mode | How it is selected | vCenter SDK behavior |
|------|-------------------|---------------------|
| Skip verify | `insecure: true` or `--insecure` | TLS verification disabled |
| Custom CA | `ca_cert` set and file exists | Trust PEM at that path |
| System CA | secure, `ca_cert` omitted | Host trust store (`ca-certificates` in image) |
| Thumbprint fallback | `fingerprint` set (any secure mode) | govmomi `SetThumbprint` when CA verification fails |

**vCenter fingerprint vs ESXi thumbprints:** `fingerprint` is the vCenter SSL
thumbprint used for vCenter SDK connections. ESXi NFC disk downloads use
separate thumbprints registered automatically from the NFC lease during export
— not `CopyInput.fingerprint`.

Progress is written to the `--output` path (default `copy-progress.json`; when
omitted in JSON input, falls back to `$WORKDIR/copy-progress.json`).

## Copy Flow

For each selected source disk → empty PVC target (up to `copy_concurrency` disks in parallel):

1. Connect to vCenter via govmomi, export VM via NFC lease
2. Filter lease URLs to disks matching `source_disks` (normalized path; list order preserved)
3. Require selected count equals empty target count
4. Per disk (worker pool): govmomi NFC download (ESXi thumbprint from lease) → `StreamToRaw` VMDK-to-raw converter → direct write to target
5. On first disk failure, cancel remaining in-flight copies
6. For block device targets, `fsync` the device before closing
7. Complete NFC lease

Count gate: `len(FilterDiskURLs(...))` must equal empty PVC count.

### Disk selection and PVC ordering

1. `source_disks` lists the VMDKs to copy.
2. `mapDiskURLs` labels each NFC disk URL with the VM backing `FileName` (normalized)
   and drops non-disk lease items (nvram, etc. via `HttpNfcLeaseDeviceUrl.disk`).
3. NFC lease items are matched to `source_disks` by normalized path
   (`disk-000001.vmdk` → `disk.vmdk`).
4. Selected disks keep **source_disks order**; lease disks not in the list are skipped.
5. Empty PVC targets are sorted by numeric index (`/dev/blockN` or `/mnt/disks/diskN`).
6. Pairing is `targets[i]` ← `selected[i]`.

Omitting a disk from `source_disks` (e.g. a shared disk) skips copying it.

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

Target paths (typical pod mounts):

| PVC volumeMode | Write path |
|----------------|------------|
| Block | `/dev/block{N}` |
| Filesystem | `/mnt/disks/disk{N}/disk.img` |

## Install paths

| Path | Role |
|------|------|
| `/usr/lib/kc-utils/kc-copy` | Default install path in container image |
| `/usr/bin/kc-copy` | Convenience path for manual debug |

Locally, `make build` places `kc-copy` in `bin/`.

## Package layout (`pkg/copy/`)

| File | Purpose |
|------|---------|
| `copy.go` | `Run()` entry point, worker pool, progress tracking |
| `download.go` | NFC HTTP download and `StreamToRaw` orchestration |
| `filter.go` | `FilterDiskURLs` — match NFC lease URLs to `source_disks` |
| `target.go` | Target PVC discovery and ordering |
| `vmdk.go` | `StreamToRaw` — stream-optimized VMDK to raw conversion |
| `vsphere.go` | govmomi vSphere connection and NFC lease setup |
| `drain_linux.go` | Block device `fsync` before close (Linux) |
| `drain_other.go` | No-op drain for non-Linux builds |

## Related

- [README.md](README.md) — four-binary pipeline overview
- [kc-v2v.md](kc-v2v.md) — orchestrator that may invoke `kc-copy` with `--input`
- [pkg/copy/README.md](../../pkg/copy/README.md) — package reference
