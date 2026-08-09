# pkg/copy — NFC disk copy

Standalone vSphere disk copy via govmomi NFC (Network File Copy) export.
Downloads VMDK disks from vCenter/ESXi over HTTPS — no VMware VDDK library
required — and writes raw disk images to PVC targets (block devices or
filesystem images).

This package implements the `kc-copy` binary's core logic. In the kc-v2v
pipeline, `kc-copy` runs as an optional first step to pull VM disks from
vSphere into local PVCs before the prepare/convert/finalize stages operate
on them.

## How it works

1. **Target discovery** — scans `/dev/block[0-9]*` (block devices) and
   `/mnt/disks/disk[0-9]*` (filesystem mounts) for conversion-pod PVCs.
   Filters to empty targets (block devices whose first 1 MiB is all zeros,
   or filesystem images smaller than 1 MiB).

2. **NFC export** — connects to vSphere using credentials from
   `/etc/secret/accessKeyId` and `/etc/secret/secretKey`, locates the VM by
   name, and starts an NFC export lease via govmomi. TLS uses the same policy
   as Forklift virt-v2v: skip verify when `insecure` is set, otherwise trust
   the CA bundle at `ca_bundle` (default `/opt/ca-bundle.crt` after
   `LinkCertificates`). The lease provides HTTPS URLs for each virtual disk.

3. **Disk matching** — filters the NFC lease URLs to only the requested
   source VMDK paths (snapshot delta suffixes like `-000001.vmdk` are
   normalized to base `.vmdk` names). Validates that the number of selected
   source disks matches the number of empty targets.

4. **Concurrent copy** — downloads disks in parallel (default concurrency 4,
   bounded by a semaphore). Each disk is streamed through `StreamToRaw`,
   which decompresses the stream-optimized VMDK format on the fly and writes
   sparse raw output (zero regions are skipped via seek).

5. **Progress tracking** — writes a `copy-progress.json` file after each disk
   completes. On failure of any disk, cancels all remaining copies and aborts
   the NFC lease.

## VMDK stream-to-raw conversion

`StreamToRaw` reads a stream-optimized VMDK (the format vSphere uses for NFC
export) and produces a raw disk image:

- Parses the 512-byte VMDK header to extract capacity, grain size, and
  overhead.
- Iterates grain markers: compressed grains are decompressed via zlib and
  written at their LBA offset; metadata markers (grain table, grain directory,
  footer) are skipped.
- Reuses a single zlib reader across grains to avoid per-grain allocation of
  the internal flate dictionary (~44 KiB each).
- Safety caps: grain size limited to 64 MiB, compressed buffer limited to 2x
  grain size.

## Page cache management (Linux)

On Linux, the `drainWriter` wrapper periodically calls `fdatasync` +
`fadvise(FADV_DONTNEED)` (every 32 MiB written) to release page cache back to
the kernel. Without this, the cgroup memory usage grows by the total amount
written (3-4 GiB for typical VM disks), which can cause OOM kills in
memory-constrained pods.

## File layout

| File | Role |
|------|------|
| `copy.go` | Entry point (`Run`), target validation, concurrent copy orchestration |
| `vmdk.go` | `StreamToRaw` — VMDK decompression and raw disk writing |
| `vsphere.go` | vSphere connection, NFC lease management, disk URL mapping |
| `download.go` | HTTP download, `CopyDisk`, progress logging |
| `filter.go` | VMDK path normalization and NFC lease filtering |
| `target.go` | PVC target discovery (`DiscoverTargets`, `EmptyTargets`) |
| `drain_linux.go` | Linux page-cache drain via fdatasync + fadvise |
| `drain_other.go` | No-op drain for non-Linux platforms |

Import path: `github.com/yaacov/kc-utils/pkg/copy`
