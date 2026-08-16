# kc-v2v Orchestrator

Drop-in replacement for Forklift's `virt-v2v` container entrypoint. Runs the
full in-place conversion pipeline inside the conversion pod: optional NFC disk
copy, then `kc-prepare` → `kc-convert-{linux,windows}` → `kc-finalize`, then
Forklift-compatible inspection XML and HTTP API.

## Entry Point

`cmd/kc-v2v/main.go` — bootstrap via `pkg/v2v/env/`; orchestration in
[`pkg/cmd/v2v/pipeline.go`](../../pkg/cmd/v2v/pipeline.go); HTTP server in
[`pkg/cmd/v2v/server/`](../../pkg/cmd/v2v/server/).

## Flow

```text
env.Load (V2V_* env + flags)          # cmd/kc-v2v bootstrap
  → LinkCertificates / EnsureWorkdir
  → v2v.Run(cfg)
      → env.NeedsCopy?  → ResolveCopySources → ValidateCopySourceCount → write copy-input.json → kc-copy
      → DiscoverDisks
      → StartSharedSession(backend): nil when backend has no SharedSessionFactory
      → kc-prepare → kc-convert-* → kc-finalize (--backend <name>)
           (guestfs: GUESTFISH_PID / KC_GUESTFISH_PID;
            qemu: KC_AGENT_SOCK / KC_QEMU_PID; pkg/guest only)
      → on failure after prepare may have set up guest:
           kc-finalize --teardown-only (best-effort backend teardown, no Sync)
      → if shared session was started: SharedSession.Close
      → WriteInspectionXML
      → HTTP server (:8080) when LOCAL_MIGRATION=true
```

Converter selection mirrors upstream virt-v2v: `PrepareOutput.converter` (or
`PrepareOutput.inspect.type`) determines `kc-convert-linux` vs
`kc-convert-windows`.

### Failure cleanup

If the pipeline fails after `kc-prepare` may have created guest state, `kc-v2v`
best-effort runs `kc-finalize --teardown-only` to reclaim orphaned resources
(mounts/LUKS/loops in direct mode; appliance mounts in guestfs mode — the
container default). That path never Syncs guest edits back to disks.

`SharedSession.Close` is separate and conditional: backends without
`SharedSessionFactory` return `nil` from `StartSharedSession`, so Close is
skipped. When a shared session was started (guestfs), its PID is still passed
into teardown-only, and Close runs afterward on success or failure.

qcow2 overlays (when enabled) are discarded separately by `RunWithOverlay`.
Partial PVC data and workdir JSON are left alone. Cleanup failures are logged
as warnings and do not replace the original pipeline error.

See [kc-finalize.md](kc-finalize.md) (`--teardown-only`).

## Orchestration Stages

| # | Stage | Package | Description |
|---|-------|---------|-------------|
| 1 | Config | `pkg/v2v/config/` | `V2V_*` environment variable schema |
| 2 | Load | `pkg/v2v/env/` | Parse env/flags, link certs, create workdir |
| 3 | Orchestrate | `pkg/cmd/v2v/pipeline.go` | Copy gate, discover, subprocess pipeline, inspection, HTTP |
| 4 | Copy | `kc-copy` (`pkg/copy/`) | Optional NFC disk copy subprocess |
| 5 | Discover | `pkg/v2v/env/disks.go` | Find disks on conversion-pod PVC mounts |
| 6 | Inspection | `pkg/v2v/inspection/xml/` | Write Forklift-compatible inspection XML |
| 7 | HTTP | `pkg/cmd/v2v/server/` | Serve `/vm`, `/inspection`, `/warnings`, `/shutdown` |

Stage 4 is skipped when disks are already populated (CDI, copy-offload, EC2,
Nutanix pre-fill, etc.). Pipeline binaries (including `kc-copy`) live under
`KC_BIN_DIR` (default `/usr/lib/kc-utils`). See [kc-copy.md](kc-copy.md) and
[pkg/v2v/README.md](../../pkg/v2v/README.md).

## Filesystem checks

kc-v2v does not run fsck itself. The prepare and finalize subprocesses call
`Guest.FSCheck()` twice per conversion: pre-fsck in `kc-prepare` (before mount)
and post-fsck in `kc-finalize` (after unmount). With the image default
`V2V_backend=guestfs`, checks run inside the libguestfs appliance using the guestfs
backend matrix (ext*, xfs, ntfs/ntfs3; btrfs is not fsck'd).

On pipeline failure after prepare may have set up guest state, kc-v2v runs
`kc-finalize --teardown-only`, which reclaims mounts/LUKS/loops but **does not**
run fsck.

Full timeline, supported filesystem types, check-vs-repair semantics, and
Windows NTFS distinctions:
[../architecture/filesystem-checks.md](../architecture/filesystem-checks.md).

## Configuration

Configuration is loaded from `V2V_*` environment variables with CLI flag
overrides via `env.Load()`. Full schema:
[`pkg/v2v/config/config.go`](../../pkg/v2v/config/config.go).

### Required

| Variable | When | Purpose |
|----------|------|---------|
| `V2V_source` | Always | Source hypervisor (`vSphere`, `ec2`, `nutanix`, …) |
| `V2V_backend` | Always | Guest disk backend (`direct`\|`guestfs`\|`qemu`; no default). Image sets `guestfs`. |
| `V2V_libvirtURL` | vSphere copy | vCenter URL (govmomi inventory + NFC export) |
| `V2V_firmware` | vSphere | Optional firmware override (`uefi` or `bios`) |
| `V2V_vmName` | vSphere copy | Source VM name |
| `V2V_fingerprint` | vSphere copy | vCenter SSL thumbprint |

### Copy vs in-place

| `V2V_inPlace` | Behavior | Expected disk state |
|---------------|----------|---------------------|
| unset (default) / `0` | Run NFC disk copy, then convert | Blank PVCs |
| `1` / `true` | Skip copy, convert in-place | Pre-filled PVCs |

`Load()` validates that PVC state matches the flag; mismatch fails early.

### Common optional

| Variable | Default | Purpose |
|----------|---------|---------|
| `V2V_inPlace` | `false` (copy) | Skip disk copy when `1` (pre-filled disks) |
| `LOCAL_MIGRATION` | `true` | Start HTTP server on `:8080` |
| `V2V_RootDisk` | `first` | Root selection policy passed to kc-prepare |
| `V2V_staticIPs` | | Static IP mapping for Windows guests |
| `V2V_overlayEnabled` | `true` | qcow2 overlay during conversion |
| `V2V_NBDE_CLEVIS` | `false` | Enable Clevis LUKS unlock (Forklift Plan `nbdeClevis` / Conversion `diskEncryption.type=Clevis`). Guestfs enables appliance networking; Tang must be reachable from the pod. Clevis takes precedence over `/etc/luks` keyfiles. |
| `V2V_offline` | `false` | Pass `--offline` to converters (use image-staged packages only) |

The container image bakes Windows virtio-win drivers under
`/usr/share/virtio-win/drivers/by-os/` and RHEL-family offline
`qemu-guest-agent` RPMs under `/usr/share/kc-packages/rpm/el{8,9,10}/x86_64/`.
Set `V2V_offline=true` to pass `--offline` to converters. See [build/kc-v2v/README.md](../build/kc-v2v/README.md).

Disk copy settings:

| Variable | Default |
|----------|---------|
| `V2V_copyConcurrency` | `4` (max parallel NFC disk copies; capped at disk count) |

### TLS (kc-v2v / Forklift)

kc-v2v uses Forklift conversion-pod signals only (no `V2V_caCert` env vars):

| Forklift signal | TLS mode |
|-----------------|----------|
| `no_verify=1` in `V2V_libvirtURL` | Insecure |
| `/etc/secret/cacert` mounted | Custom CA |
| Secure, no provider CA in secret | System CA |

At startup, kc-v2v calls `LinkCertificates`: when `/etc/secret/cacert` exists,
symlinks `/opt/ca-bundle.crt` → secret (Forklift virt-v2v parity).

When disk copy runs, `env.BuildCopyInput` writes `copy-input.json` for
`kc-copy`. That maps Forklift TLS state into kc-copy's two input fields
(`kc-copy` does not interpret Forklift signals itself):

| Forklift state | `copy-input.json` |
|----------------|-------------------|
| `no_verify=1` in `V2V_libvirtURL` | `"insecure": true` |
| `/etc/secret/cacert` mounted | `"ca_cert": "/etc/secret/cacert"` |
| Secure, no provider CA in secret | omit both → kc-copy uses system CA |

Workdir defaults: `/var/tmp/v2v` (JSON handoff files), mount root `/tmp/kc-guest`.

### QEMU appliance (`V2V_backend=qemu`)

| Variable | Purpose |
|----------|---------|
| `KC_AGENT_SOCK` | Unix socket for `kc-agent` (set by `kc-v2v` shared session) |
| `KC_QEMU_PID` | QEMU pid after prepare Setup (liveness) |
| `KC_APPLIANCE_DIR` | Directory with `vmlinuz` and `initramfs.img` for host `GOARCH` |
| `KC_VIRTIO_WIN` | Host virtio-win tree (same as other backends; not packed in the appliance) |
| `KC_PACKAGES` | Host qemu-ga package tree (same as other backends) |
| `KC_GUESTFS_NETWORK` | `1`/`true` enables QEMU user-net for Clevis (same env as guestfs) |

Build artifacts with `make appliance`. On macOS install Homebrew QEMU; virtio-win
and qemu-ga RPMs stay on the host (`KC_VIRTIO_WIN` / `KC_PACKAGES`). Stage those
trees with `make stage-offline` (Fedora container → gitignored `build/offline/`).
The Mac does not need hivex, libguestfs, or LVM.

## HTTP API

When `LOCAL_MIGRATION=true`, serves Forklift-compatible endpoints on `:8080`:

| Endpoint | Response |
|----------|----------|
| `GET /vm` | 204 No Content (in-place mode) |
| `GET /inspection` | Inspection XML file |
| `GET /warnings` | JSON warnings array, or 204 when empty |
| `GET /shutdown` | Graceful server shutdown |

## Forklift Wiring

Set the virt-v2v image FQIN to the kc-v2v container:

```bash
oc mtv settings set --setting virt_v2v_image_fqin --value quay.io/you/kc-v2v:latest
```

The migration controller starts the conversion pod with
`ENTRYPOINT ["/usr/bin/kc-v2v"]`. Disk copy uses pure Go govmomi NFC export
(no external libraries required).

| kc-v2v path | When | Behavior |
|---|---|---|
| Copy + convert | `V2V_inPlace` unset/`0` (default) | Spawn `kc-copy`, then in-place convert |
| Convert only | `V2V_inPlace=1` | `DiscoverDisks()` on attached volumes, then convert |

Pre-filled PVCs appear as `/dev/blockN` or `/mnt/disks/diskN/disk.img` in the
pod — that is normal Kubernetes volume attachment, not a separate path per
source (EC2, CDI, Nutanix, etc.). Hypervisor cleanup runs in convert plugins
from `V2V_source`, same as every other source.

Container build and Forklift Plan configuration:
[build/kc-v2v/README.md](../build/kc-v2v/README.md).

### Avoid unsupported Forklift features

kc-v2v replaces in-place virt-v2v pods only. Configure Forklift so other workflows
do not expect upstream virt-v2v:

- Set `virt_v2v_image_fqin` to the kc-v2v image; unset `virt_v2v_extra_args`
  (kc-v2v ignores `V2V_extra_args`).
- Keep `skipGuestConversion: false` on Plans.
- For warm vSphere: set `runPreflightInspection: false` to skip virt-v2v-inspector
  (default runs a separate image before disk transfer).
- Do not use `Conversion` CRs with type `Remote`, `DeepInspection`, or
  `Inspection` — use Plan migrations or `InPlace` conversions.

Full configuration table and YAML examples:
[build/kc-v2v/README.md — Forklift configuration for kc-v2v](../build/kc-v2v/README.md#forklift-configuration-for-kc-v2v).

## Supported vs Unsupported

| Supported | Unsupported |
|-----------|-------------|
| In-place EC2 / copy-offload / Nutanix (pre-filled) | Remote copy (`Conversion` type `Remote`) |
| vSphere NFC disk copy into blank PVCs | Deep inspection, virt-v2v-inspector preflight |
| qcow2 overlay (`V2V_overlayEnabled`) | `V2V_extra_args` / `virt_v2v_extra_args` passthrough |

## Related

- [kc-copy.md](kc-copy.md) — NFC disk copy stage CLI (subprocess + standalone)
- [README.md](README.md) — four-binary pipeline overview
- [pkg/v2v/README.md](../../pkg/v2v/README.md) — kc-v2v package internals
