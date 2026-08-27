# kc-copy Utility

Standalone vSphere NFC disk copy: downloads VMDK disks from vCenter/ESXi over
HTTPS and writes raw images to empty PVC targets (block devices or filesystem
images) or to `{target_dir}/diskN.img`. Pure Go govmomi export.

Runs [`pkg/copy/`](../../pkg/copy/) via `copy.Run(CopyInput)`.

**Entry:** `cmd/kc-copy/main.go` — flags or `--input` JSON; no subcommands.

Compiles and runs on any Unix host (no guest disk backend).

## CLI

Two input modes:

1. **`--input copy-input.json`** — read `CopyInput` directly
2. **Flags** — `--host`, `--username`, `--password`, `--vm-name`, `--fingerprint`, `--target-dir`, `--disk-path`, TLS flags, etc.

Exit codes: `0` success, `1` configuration or copy error.

## Usage

```bash
make build-kc-copy

kc-copy \
  --host vcenter.example.com \
  --username myuser \
  --password mypass \
  --datacenter mydatacenter \
  --ca-cert /path/to/ca.pem \
  --vm-name my-vm \
  --fingerprint "AB:CD:..." \
  --target-dir /data/my-vm
# writes /data/my-vm/disk0.img, disk1.img, … (all VM disks)

kc-copy \
  --host vcenter.example.com \
  --username myuser \
  --password mypass \
  --datacenter mydatacenter \
  --ca-cert /path/to/ca.pem \
  --vm-name my-vm \
  --fingerprint "AB:CD:..." \
  --disk-path "[ds] vm/disk1.vmdk,[ds] vm/disk2.vmdk" \
  --output copy-progress.json
```

JSON input (skip TLS verification; write images under `target_dir`):

```bash
cat > copy-input.json <<'EOF'
{
  "host": "vcenter.example.com",
  "datacenter": "mydatacenter",
  "username": "myuser",
  "password": "mypass",
  "insecure": true,
  "vm_name": "my-vm",
  "fingerprint": "...",
  "target_dir": "/data/my-vm",
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

`source_disks` / `--disk-path` selects which NFC lease disks to copy. Omit it
to copy every disk on the VM. `--target-dir` / `target_dir` writes
`diskN.img` files under that directory instead of discovering PVC targets.

## Configuration

| Flag / JSON field | Default | Description |
|-------------------|---------|-------------|
| `--host` / `host` | (required) | vCenter or ESXi hostname |
| `--datacenter` / `datacenter` | — | vSphere datacenter name (optional) |
| `--username` / `username` | — | vSphere username; empty falls back to `/etc/secret/accessKeyId` |
| `--password` / `password` | — | vSphere password; empty falls back to `--password-file`, then `/etc/secret/secretKey` |
| `--password-file` | — | Password file. Used when `--password` and JSON `password` are empty |
| `--insecure` / `insecure` | `false` | Skip TLS certificate verification |
| `--ca-cert` / `ca_cert` | — | PEM custom CA path; error if set but file missing |
| `--vm-name` / `vm_name` | (required) | Source VM name |
| `--fingerprint` / `fingerprint` | (required) | vCenter SSL thumbprint fallback for SDK TLS |
| `--disk-path` / `source_disks` | (optional) | VMDKs to select from the NFC lease; omit to copy all disks |
| `--target-dir` / `target_dir` | — | Write `{target_dir}/diskN.img` instead of PVC targets |
| `--work-dir` / `workdir` | `/var/tmp/v2v` | Working directory (progress JSON, not disk images) |
| `--output` / `output_path` | `copy-progress.json` | Progress output file |
| `--copy-concurrency` / `copy_concurrency` | `4` | Max parallel disk copies (capped at disk count; `1` = sequential) |
| `--log-level` | `info` | Log level (`debug`, `info`, `warn`, `error`) |

vSphere credentials (resolved at connect time):

1. `--username` / `--password` (or JSON `username` / `password`); `--password-file` if the password is still empty
2. `/etc/secret/accessKeyId` and `/etc/secret/secretKey` for any field still empty (Forklift secret mount)

Omit the flags in a conversion pod; the secret files are enough.

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

For each selected source disk → target (up to `copy_concurrency` disks in parallel):

1. Connect to vCenter via govmomi, export VM via NFC lease
2. Filter lease URLs to disks matching `source_disks` (normalized path; list order preserved). Empty `source_disks` selects every lease disk in lease order.
3. PVC mode: require selected count equals empty target count. `--target-dir` mode: write `{target_dir}/disk{N}.img` (`N` = 0..n-1) and skip PVC discovery.
4. Per disk (worker pool): govmomi NFC download (ESXi thumbprint from lease) → `StreamToRaw` VMDK-to-raw converter → direct write to target
5. On first disk failure, cancel remaining in-flight copies
6. For block device targets, `fsync` the device before closing
7. Complete NFC lease

Count gate (PVC mode only): `len(FilterDiskURLs(...))` must equal empty PVC count.

### Disk selection and PVC ordering

1. `source_disks` lists the VMDKs to copy, or is omitted to copy all lease disks.
2. `mapDiskURLs` labels each NFC disk URL with the VM backing `FileName` (normalized)
   and drops non-disk lease items (nvram, etc. via `HttpNfcLeaseDeviceUrl.disk`).
3. NFC lease items are matched to `source_disks` by normalized path
   (`disk-000001.vmdk` → `disk.vmdk`).
4. Selected disks keep **source_disks order**; lease disks not in the list are skipped.
5. Empty PVC targets are sorted by numeric index (`/dev/blockN` or `/mnt/disks/diskN`).
6. Pairing is `targets[i]` ← `selected[i]`. With `--target-dir`, `targets[i]` is
   `{target_dir}/disk{i}.img`.

Omitting a disk from an explicit `source_disks` list (e.g. a shared disk) skips
copying it. Omitting the list entirely copies every disk.

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
| `--target-dir` | `{target_dir}/disk{N}.img` |

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
| `target.go` | Target PVC discovery and `TargetsFromDir` |
| `vmdk.go` | `StreamToRaw` — stream-optimized VMDK to raw conversion |
| `vsphere.go` | govmomi vSphere connection and NFC lease setup |
| `drain_linux.go` | Block device `fsync` before close (Linux) |
| `drain_other.go` | No-op drain for non-Linux builds |

## Related

- [README.md](README.md) — four-binary pipeline overview
- [kc-v2v.md](kc-v2v.md) — orchestrator that may invoke `kc-copy` with `--input`
- [pkg/copy/README.md](../../pkg/copy/README.md) — package reference
