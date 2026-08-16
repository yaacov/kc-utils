# kc-v2v Container Image

`kc-v2v` is a drop-in replacement for Forklift's `virt-v2v` image, running the kc-utils pipeline (`kc-prepare` → `kc-convert-*` → `kc-finalize`).

The image is built with the official **golang** image for compilation and
**Fedora** (`quay.io/fedora/fedora:44`) for the runtime.

Binary documentation: [docs/apps/kc-v2v.md](../../docs/apps/kc-v2v.md).

When Forklift attaches **blank PVCs** (cold vSphere, `useV2vForTransfer` path), kc-v2v optionally spawns **`kc-copy`** for NFC disk copy before conversion. Disk copy uses pure Go govmomi NFC export (no VDDK required).

See [pkg/v2v/README.md](../../pkg/v2v/README.md) for copy selection logic, vsphere inventory, configuration, and flow details.

## Build

```bash
make build-kc-v2v-image
make test-kc-v2v-image                 # RPMs + NTFS; FS mounts if /dev/kvm
REQUIRE_GUESTFS=1 make test-kc-v2v-image  # require kvm + ext4/xfs/btrfs/ntfs mounts
```

VirtIO-Win drivers (modern + legacy pre–Win 8) are downloaded from the public
[Fedora People archive](https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/)
during image build. No manual staging required.

For local testing without the image (macOS, or a host without the `virtio-win`
RPM), stage the same trees into gitignored `build/offline/`:

```bash
make stage-offline
export KC_VIRTIO_WIN=$PWD/build/offline/virtio-win
export KC_PACKAGES=$PWD/build/offline/kc-packages
```

Override ISO URLs at build time for version bumps or mirrors:

```bash
podman build \
  --build-arg VIRTIO_WIN_MODERN_URL=https://example.com/virtio-win-0.1.285.iso \
  --build-arg VIRTIO_WIN_LEGACY_URL=https://example.com/virtio-win-0.1.160.iso \
  -f build/kc-v2v/Containerfile .
```

Or build binaries only:

```bash
make build-kc-v2v
make build-kc-copy   # copy stage binary (also used standalone)
```

### Push / release tags

`make push-kc-v2v-image` always pushes the primary tag (`REGISTRY_TAG`, default
`devel-amd64`). When HEAD is exactly on a git tag, it also tags and pushes that
version (e.g. `v0.1.0-amd64`):

```bash
git tag v0.1.0
make push-kc-v2v-image   # quay.io/you/kc-v2v:devel-amd64 + :v0.1.0-amd64
```

Point MTV at the versioned FQIN for a release:
`quay.io/you/kc-v2v:v0.1.0-amd64`.

## Image contents (guest conversion assets)

Besides kc-utils binaries and host tools (`qemu-img`, `cryptsetup`, `hivex`/`perl-hivex`, …), the image bakes:

| Path | Source | Purpose |
|------|--------|---------|
| `/usr/share/virtio-win/drivers/by-os/` | [`stage-virtio-win.sh`](stage-virtio-win.sh) — modern ISO (0.1.285) + legacy ISO (0.1.160) | Windows VirtIO drivers (all versions including pre–Win 8) |
| `/usr/share/virtio-win/guest-agent/` | same script (modern ISO) | qemu-ga MSIs |
| `/usr/share/kc-packages/rpm/el{8,9,10}/x86_64/` | [`stage-linux-packages.sh`](stage-linux-packages.sh) | Offline Linux `qemu-guest-agent` for RHEL-family guests |

`kc-v2v` passes `--offline` to converters when `V2V_offline=true`, so airgapped pods use these paths (no guest network install for QGA when a local RPM matches).

### Pre–Win 8 virtio-win OS dirs

Build stage [`stage-virtio-win.sh`](stage-virtio-win.sh) downloads a
legacy ISO (virtio-win **0.1.160**) and extracts pre–Win 8 by-os dirs
(`2k8`, `2k3`, `xp`, `vista`) that are missing from the modern ISO.

Each version handler maps to a primary by-os dir via `DriverOSPreferences()`.
There is no arbitrary cross-version fallback — except **`win2008`**, which
uses `2k8R2` when `2k8` is absent.

| Version handler | by-os dir |
|-----------------|-----------|
| `win2008` | `2k8` (fallback `2k8R2`) |
| `win2003` | `2k3` |
| `winxp` | `xp` |
| `winvista` | `vista` |
| `win2008r2` | `2k8r2` |
| `win7` | `w7` |
| `win10` | `w10` |
| `win11` | `w11` |

No `VIRTIO_WIN` env path is used at runtime — directory-only driver lookup.

### Configurable download URLs

| Build ARG / Env var | Default | Purpose |
|---------------------|---------|---------|
| `VIRTIO_WIN_MODERN_URL` | `https://fedorapeople.org/.../virtio-win-0.1.285-1/virtio-win-0.1.285.iso` | Modern drivers (Win7+, 2008R2+) |
| `VIRTIO_WIN_LEGACY_URL` | `https://fedorapeople.org/.../virtio-win-0.1.160-1/virtio-win-0.1.160.iso` | Legacy drivers (2k8, 2k3, xp, vista) |
| `VIRTIO_WIN_SKIP_DOWNLOAD` | `0` | Set to `1` to skip download (pre-populated tree) |

No VDDK sidecar or external libraries are needed — disk copy is pure Go.

## Forklift wiring

Point the migration controller at this image — same setting as virt-v2v:

```bash
oc mtv settings set --setting virt_v2v_image_fqin --value quay.io/you/kc-v2v:latest
```

The conversion pod starts with:

```text
ENTRYPOINT ["/usr/bin/kc-v2v"]
```

Pipeline stage binaries (`kc-prepare`, `kc-convert-*`, `kc-finalize`, `kc-copy`)
are installed under `/usr/lib/kc-utils/`. `kc-copy` is also at `/usr/bin/kc-copy`
for standalone debug use.

Disk copy uses govmomi NFC export — no VDDK sidecar init container is needed.

## Forklift configuration for kc-v2v

kc-v2v replaces only the **in-place virt-v2v conversion pod** (`Conversion` type `InPlace`). Forklift still runs other images for disk transfer (CDI, copy-offload), inspection, and assessment. Configure Forklift so those other paths do not expect upstream virt-v2v behavior that kc-v2v does not implement.

### Cluster settings

```bash
# Point virt-v2v pods at kc-v2v
oc mtv settings set --setting virt_v2v_image_fqin --value quay.io/you/kc-v2v:latest

# kc-v2v ignores extra virt-v2v CLI args — remove cluster-wide overrides
oc mtv settings unset --setting virt_v2v_extra_args
```

kc-v2v uses pure Go NFC export for disk copy — no VDDK init image is required.

### Plan settings

Use normal **cold** or **warm** Plans (or **`type: conversion`** when disks are already on PVCs). Guest conversion must stay enabled.

| Plan field | Value for kc-v2v | Why |
|---|---|---|
| `skipGuestConversion` | `false` (default) | `true` skips virt-v2v/kc-v2v entirely (raw copy only) |
| `runPreflightInspection` | `false` for warm vSphere | Default `true` runs preflight before disk transfer. With `FEATURE_USE_CONVERSION_CR=true` (default), Forklift creates a `DeepInspection` CR using `deep_inspection_image_fqin`. With the legacy path, it runs a `virt-v2v-inspection` pod using `virt_v2v_image_fqin` in remote inspector mode. kc-v2v supports neither. |
| `type` | `cold`, `warm`, or `conversion` | In-place conversion on attached PVCs |
| `virtV2vImage` | omit, or same kc-v2v FQIN | Per-plan override; omit to use cluster `virt_v2v_image_fqin` |

Example warm Plan with preflight inspection disabled:

```yaml
apiVersion: forklift.konveyor.io/v1beta1
kind: Plan
metadata:
  name: my-warm-plan
  namespace: openshift-mtv
spec:
  type: warm
  skipGuestConversion: false
  runPreflightInspection: false   # skip warm preflight (DeepInspection or legacy inspector)
  # ... provider, map, targetNamespace, vmSelector, etc.
```

Cold Plans do not run preflight inspection. Warm Plans run it by default when `skipGuestConversion` is false.

### Warm preflight inspection

Warm vSphere plans with `runPreflightInspection: true` (default) and
`skipGuestConversion: false` run a **PreflightInspection** step after the
initial snapshot and before disk transfer. kc-v2v is not involved in either
implementation:

| Forklift path | When | Image | Pod behavior |
|---|---|---|---|
| **DeepInspection** (default) | `FEATURE_USE_CONVERSION_CR=true` | `deep_inspection_image_fqin` | Inspects the warm snapshot via vm-migration-detective + VDDK |
| **Legacy inspection** | `FEATURE_USE_CONVERSION_CR=false` | Same as `virt_v2v_image_fqin` | Runs `virt-v2v-inspector` remotely against parent disk paths (`V2V_remoteInspection=true`) |

kc-v2v does not implement remote preflight (`V2V_remoteInspection`) and is not
the deep-inspection image. Set `runPreflightInspection: false` (or
`--run-preflight-inspection false`) on warm Plans.

**Standalone deep inspection** (UI assess flow or a manually created
`DeepInspection` Conversion CR) uses the same `deep_inspection_image_fqin`
image but is separate from the guest-conversion pod. Do not create standalone
`DeepInspection` CRs when you intend to use kc-v2v for conversion only.

### Conversion CR types

| `Conversion.spec.type` | Uses kc-v2v? | Notes |
|---|---|---|
| `InPlace` | Yes | Normal guest conversion on attached PVCs |
| `Remote` | No | Remote disk copy (`virt-v2v -o kubevirt`) — not supported |
| `DeepInspection` | No | Separate deep-inspection image |
| `Inspection` | No | Legacy virt-v2v-inspector (backward compatibility) |

Use **`InPlace`** (or Plan-driven migrations that create in-place virt-v2v pods). Avoid **`Remote`**, **`DeepInspection`**, and standalone **`Inspection`** CRs when `virt_v2v_image_fqin` points at kc-v2v.

### What kc-v2v does not handle

These are Forklift controller concerns or upstream-only virt-v2v modes — misconfiguring them causes failures unrelated to kc-v2v conversion logic:

| Feature | Forklift setting / trigger | kc-v2v behavior |
|---|---|---|
| Remote disk copy | `Conversion` type `Remote` | Unsupported — use in-place + CDI/copy-offload/blank-PVC copy |
| Deep inspection | `Conversion` type `DeepInspection`, UI assess | Different image — not kc-v2v |
| Warm preflight | Plan `runPreflightInspection: true` (default) | DeepInspection CR (default) or legacy remote `virt-v2v-inspector` pod — neither is kc-v2v |
| Extra virt-v2v args | `virt_v2v_extra_args` → `V2V_extra_args` | Ignored (warning logged) |
| Raw copy without conversion | Plan `skipGuestConversion: true` | kc-v2v never runs |

Plan **customization scripts**, **static IPs**, **LUKS**, and **feature flags** (`V2V_vsphereVmwareDriverRemoval`, etc.) are supported — see [Customization](#customization) below.

## Copy + convert flow

```text
Conversion pod (blank PVCs)
  → kc-v2v
    → v2v.Run(cfg)
      → env.NeedsCopy?  (!V2V_inPlace; default copy)
        → yes: NFC disk copy via govmomi
        → no:  skip (V2V_inPlace=1)
      → in-place conversion pipeline
```

| `V2V_inPlace` | Behavior | Expected disk state |
|---|---|---|
| unset (default) / `0` | Run NFC disk copy, then convert | Blank PVCs |
| `1` / `true` | Skip copy, convert in-place | Pre-filled PVCs |

`ValidateCopyMode` fails if the flag and PVC state disagree.

kc-v2v does not distinguish CDI, EC2, Nutanix, copy-offload, etc. — those differ
only in how Forklift fills PVCs before the pod starts. Attached block PVCs are
always discovered at `/dev/blockN`.

Forklift does not yet ship a Nutanix provider — disk copy must happen before
kc-v2v runs. Example:

```bash
V2V_inPlace=1 V2V_source=nutanix V2V_vmName=my-vm kc-v2v
```

## Requirements

- **Blank PVC / copy path (default)** — unset or `V2V_inPlace=0`; empty PVCs
- **Pre-filled disks** — `V2V_inPlace=1` (CDI populator, offload, etc.)
- **NFC disk copy** — requires `V2V_libvirtURL`, `V2V_fingerprint`, `V2V_vmName`, and vSphere credentials

## Environment variables

| Variable | Purpose |
|---|---|
| `V2V_inPlace` | Skip copy when `1`; default unset/`0` runs NFC disk copy |
| `V2V_libvirtURL` | vCenter connection URL (govmomi NFC export) |
| `V2V_firmware` | Optional guest firmware override (`uefi` or `bios`) |
| `V2V_fingerprint` | vCenter SSL thumbprint |
| `V2V_vmName` | Source VM name |
| `V2V_backend` | Guest disk backend (`direct`\|`guestfs`\|`qemu`). Image sets `guestfs` explicitly |

### Guestfs (default in image)

With `V2V_backend=guestfs`, `kc-v2v` starts one `guestfish --listen`
appliance and passes `GUESTFISH_PID` into prepare/convert/finalize so discovery,
file I/O, trim, and fsck reuse the same QEMU. Fsck runs inside the appliance
during `kc-prepare` (pre-fsck, before mount) and `kc-finalize` (post-fsck, after
unmount) — see
[docs/architecture/filesystem-checks.md](../../docs/architecture/filesystem-checks.md).
Guest filesystems stay mounted inside the appliance during convert; stages read
and write via guestfish RPC (`Checkout` for tools that need a host path, e.g.
hivex). Set `V2V_backend=direct` for privileged host mounts.

The UBI image omits host-side e2fsprogs, xfsprogs, btrfs-progs, and ntfs-3g
because fsck runs in the appliance VM, not on the conversion-pod host. See
[build/kc-v2v/ubi/Containerfile](ubi/Containerfile).

Fedora libguestfs mounts NTFS with plain `guestfish`. On RHEL/CentOS/UBI,
libguestfs only allows NTFS when the program name starts with `virt-`;
kc-utils prefers `virt-guestfish` when that binary is present. Details:
[docs/architecture/privilege-model.md](../../docs/architecture/privilege-model.md#ntfs-mounts-on-rhelcentosubi).

See also existing `V2V_*` flags in [`pkg/v2v/config/config.go`](../../pkg/v2v/config/config.go).

## Customization

| Mechanism | Handling |
|---|---|
| Plan `customizationScripts` ConfigMap | Mounted at `/mnt/dynamic_scripts`; processed by kc-finalize `dynamicscriptslinux` / `dynamicscriptswindows` plugins |
| `V2V_staticIPs` | Parsed into `PrepareInput.options.static_ips` |
| `V2V_HOSTNAME` | Set via finalize `native` customizer |
| `V2V_NBDE_CLEVIS` / `/etc/luks/*` | Clevis unlock (`clevis-luks-unlock` in guestfs; host `clevis` in direct) or keyfiles from `/etc/luks`. Clevis wins if both are set. Tang must be reachable when Clevis is enabled. |
| Feature flags (`V2V_vsphereVmwareDriverRemoval`, etc.) | Mapped to converter pipeline options |

### Dynamic script filename patterns

| OS | Pattern | Example |
|---|---|---|
| Linux run | `NN_linux_run_*.sh` | `05_linux_run_install-agent.sh` |
| Linux firstboot | `NN_linux_firstboot_*.sh` | `10_linux_firstboot_config.sh` |
| Windows firstboot | `NN_win_firstboot_*.ps1` | `20_win_firstboot_join-domain.ps1` |

Non-matching filenames are skipped.

## HTTP API (local migration)

When `LOCAL_MIGRATION=true`, kc-v2v serves Forklift-compatible endpoints on `:8080`:

| Endpoint | Response |
|---|---|
| `GET /vm` | 204 No Content (in-place) |
| `GET /inspection` | `/var/tmp/v2v/inspection.xml` |
| `GET /warnings` | JSON warnings or 204 |
| `GET /shutdown` | Shuts down the server |

## Supported vs unsupported

See [Forklift configuration for kc-v2v](#forklift-configuration-for-kc-v2v) for Plan and Conversion CR settings that avoid unsupported workflows.

| Supported | Unsupported |
|---|---|
| In-place EC2 / copy-offload / Nutanix (pre-filled) | Remote copy (`virt-v2v -o kubevirt`, `Conversion` type `Remote`) |
| vSphere NFC disk copy into blank PVCs (govmomi NFC export) | N/A |
| govmomi vCenter inventory (disks, NICs, firmware) | Deep inspection (`Conversion` type `DeepInspection`) |
| qcow2 overlay safety (`V2V_overlayEnabled`) | Warm preflight (Plan `runPreflightInspection: true` — DeepInspection or legacy inspector) |
| Plan customization scripts, static IPs, LUKS | `V2V_extra_args` / `virt_v2v_extra_args` passthrough |
| | Raw copy without conversion (`skipGuestConversion: true`) |
