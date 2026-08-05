# Windows virtio-win vendor files

The modern virtio-win 1.9.40 RPM does not ship `by-os` trees for pre–Win 8
guests. The kc-v2v image can stage those **per Windows version** (like
`rpm/el8`, `el9`, `el10` for Linux QGA):

| by-os dir | Version handler |
|-----------|-----------------|
| `2k8` | `win2008` |
| `2k3` | `win2003` |
| `xp` | `winxp` |
| `vista` | `winvista` |

These files are not committed to git. **Image build succeeds without them** —
staging is best-effort when an artifact is present. Pre–Win 8 guest conversion
requires the dirs below; otherwise conversion fails at runtime with a hint
pointing here.

## No open/free public source

Pre–Win 8 driver trees are **not** available from public CentOS/Koji virtio-win
RPMs (el8+ builds strip `2k8`/`2k3`/`xp` at packaging time) or from the
virtio-win git repos (build scripts only, no signed binaries).

The known-good artifact is **virtio-win 1.9.12-4.el7** (RPM or ISO) — the same
package Forklift downstream virt-v2v uses. It is published on the RHEL
supplementary channel and requires RHEL entitlement to download.

## Optional staging (when you have the artifact)

| File | How to provide |
|------|----------------|
| `virtio-win-1.9.12-4.el7.noarch.rpm` | Copy into this directory, or set `VIRTIO_WIN_RPM` |
| `virtio-win-1.9.12.iso` | Copy into this directory, or set `VIRTIO_WIN_ISO` |

Then run (optional — not required for `make build-kc-v2v-image`):

```bash
make prepare-windows-virtio-drivers
make build-kc-v2v-image
```

Examples:

```bash
cp /path/to/virtio-win-1.9.12-4.el7.noarch.rpm build/kc-v2v/vendor/
make prepare-windows-virtio-drivers

VIRTIO_WIN_RPM=/path/to/virtio-win-1.9.12-4.el7.noarch.rpm make prepare-windows-virtio-drivers
```

Optional environment variables for automation:

| Variable | Purpose |
|----------|---------|
| `VIRTIO_WIN_RPM` | Path to the virtio-win RPM to copy into `vendor/` |
| `VIRTIO_WIN_ISO` | Path to a virtio-win ISO to copy into `vendor/` |
| `FORKLIFT_ROOT` | Forklift clone: copy from its `build/*/vendor/` or build winlegacyiso |
| `FORKLIFT_VIRT_V2V_IMAGE` | Extract ISO from an existing Forklift virt-v2v image |
| `PREPARE_VIRTIO_WIN_STRICT=1` | Fail `make prepare-windows-virtio-drivers` when no artifact is found (default: warn and skip) |
