# kc-v2v — pkg libraries

Importable libraries for the v2v orchestrator ([`cmd/kc-v2v`](../../cmd/kc-v2v/main.go)).
Orchestration lives in [`internal/v2v/`](../../internal/v2v/).

| Package | Description |
|---------|-------------|
| [`config/`](config/) | `V2V_*` environment variable schema |
| [`env/`](env/) | Config load, disk discovery, prepare-input builders |
| [`../copy/`](../copy/) | NFC disk copy for vSphere blank PVCs (standalone package) |
| [`vsphere/`](vsphere/) | vCenter inventory via govmomi (disks, NICs, firmware) |
| [`inspection/xml/`](inspection/xml/) | Forklift-compatible inspection XML |

Import path prefix: `github.com/yaacov/kc-utils/pkg/v2v/…`

Documentation: [docs/kc-v2v.md](../../docs/kc-v2v.md),
[docs/kc-copy.md](../../docs/kc-copy.md).

## Pipeline

```text
env.Load (V2V_* env + flags)          # cmd/kc-v2v bootstrap
  → LinkCertificates / EnsureWorkdir
  → internal/v2v.Run(cfg)
      → env.NeedsCopy?  → ResolveCopySources → write copy-input.json → kc-copy
      → env.DiscoverDisks
      → kc-prepare → kc-convert-* → kc-finalize
      → inspection/xml.WriteInspectionXML
      → HTTP server (:8080) when LOCAL_MIGRATION=true
```

| kc-v2v path | When | Behavior |
|---|---|---|
| Copy + convert | `V2V_inPlace` unset/`0` (default) | `env.NeedsCopy()` → spawn `kc-copy`, then pipeline |
| Convert only | `V2V_inPlace=1` | `DiscoverDisks()` → pipeline |

Block-mode PVCs show up in the pod as `/dev/blockN`; filesystem-mode PVCs as
`/mnt/disks/diskN/disk.img`. That is how Kubernetes exposes attached volumes,
not a per-source code path.

Hypervisor cleanup (VMware tools, EC2 agents, Nutanix NGT, Xen drivers, …) runs
in the convert plugins from `V2V_source` and guest OS — not in kc-v2v itself.

Internal handoff between stages uses JSON files under `/var/tmp/v2v`
(`copy-input.json`, `prepare-input.json`, `prepare-out.json`, etc.). The only
XML output is `inspection.xml` for Forklift `GET /inspection`.

---

## config — environment schema

Defines Forklift-compatible `V2V_*` constant names and the `Config` struct
consumed by all other packages.

| File | Role |
|---|---|
| `config.go` | Env var names, path defaults, `Config` struct |

Key defaults:

| Constant | Value |
|---|---|
| `BlockGlob` | `/dev/block[0-9]*` |
| `FSGlob` | `/mnt/disks/disk[0-9]*` |
| `DefaultWorkdir` | `/var/tmp/v2v` |
| `DefaultInspectionOutputFile` | `/var/tmp/v2v/inspection.xml` |

Full variable list: [`config/config.go`](config/config.go). User-facing table:
[docs/kc-v2v.md](../../docs/kc-v2v.md#configuration).

---

## env — config and prepare input

Loads configuration, discovers attached disks, and builds `PrepareInput` for
kc-prepare. Called by [`internal/v2v/pipeline.go`](../../internal/v2v/pipeline.go).

### Load and setup

| Function | File | Role |
|---|---|---|
| `Load()` | `load.go` | Parse `V2V_*` env vars and CLI flags |
| `LinkCertificates()` | `load.go` | Symlink vSphere CA to `V2V_caBundle` (default `/opt/ca-bundle.crt`) |
| `EnsureWorkdir()` | `load.go` | Create `/var/tmp/v2v` |
| `NeedsCopy()` | `copy.go` | Flag-only: `!IsInPlace` (default copy) |
| `ValidateCopyMode()` | `copy.go` | Fail if PVC state mismatches the flag |
| `ResolveCopySources()` | `copy.go` | Ordered vmdk paths for disk copy |
| `BuildCopyInput()` | `copy.go` | Build `copy.CopyInput` JSON for `kc-copy` |

`Load()` calls `ValidateCopyMode()`. Copy vs skip is decided only by
`V2V_inPlace` (default unset/`false` = copy). Heuristics (empty PVCs, source
type) must match the flag or Load fails. After a successful copy, the pipeline
sets in-place mode for the convert/HTTP stage.

### Disk discovery

| Function | File | Role |
|---|---|---|
| `DiscoverDisks()` | `disks.go` | Glob block devices and filesystem disk images |
| `ToOverlayDisks()` | `disks.go` | Convert paths for qcow2 overlay layer |

Discovery order:

1. `/dev/block[0-9]*` — block-mode PVCs (Forklift `volumeDevices`)
2. `/mnt/disks/disk[0-9]*/disk.img` — filesystem-mode PVCs

Disks are sorted numerically (`block0` before `block1`). Pre-filled sources
use discovery only — disk paths are not passed via env vars (except optional
`V2V_diskPath` for HyperV/OVA).

### Source metadata

| Function | File | Role |
|---|---|---|
| `FetchSourceMeta()` | `sourcemeta.go` | NIC MACs, firmware hint, hostname |
| `BuildPrepareInput()` | `build.go` | Assemble `types.PrepareInput` for kc-prepare |
| `BuildLUKSSpec()` | `luks.go` | LUKS keys from `/etc/luks` or Clevis |
| `ParseStaticIPs()` | `staticip.go` | Parse `V2V_staticIPs` for Windows guests |

For vSphere, `FetchSourceMeta()` calls `vsphere.LoadInventory()` when
`V2V_libvirtURL` and `V2V_vmName` are set. `V2V_firmware` overrides the
vCenter firmware hint. Other sources (EC2, Nutanix, OVA) use env vars only;
firmware defaults to `bios` when unset.

The `env` package re-exports `config` constants via [`alias.go`](env/alias.go)
so callers can import a single package.

---

## copy — NFC disk copy (standalone package)

The disk copy package has been moved to [`pkg/copy/`](../copy/) as a first-class
standalone package. It uses pure Go govmomi NFC export (no VDDK required).

### Copy gate and source resolution (in `env/`)

| Function | File | Role |
|---|---|---|
| `NeedsCopy()` | `copy.go` | `!IsInPlace` — flag only (default copy) |
| `ValidateCopyMode()` | `copy.go` | Heuristics must match flag or fail |
| `ResolveCopySources()` | `copy.go` | `V2V_diskPath` or `vsphere.LoadInventory` |
| `BuildCopyInput()` | `copy.go` | Map config + disks → `copy.CopyInput` |
| `IsVSphereSource()` | `copy.go` | Source type check |

| `V2V_inPlace` | Behavior | Expected disk state |
|---|---|---|
| unset (default) / `0` | Run NFC disk copy, then convert | Blank PVCs |
| `1` / `true` | Skip copy, convert in-place | Pre-filled PVCs |

`ValidateCopyMode` fails when the flag and disk state disagree (e.g.
copy requested but PVCs already populated, or `V2V_inPlace=1` with empty PVCs).

### Integration

```text
kc-v2v
  → internal/v2v.Run
      → env.NeedsCopy? (!V2V_inPlace) → ResolveCopySources → kc-copy (NFC)
      → DiscoverDisks → kc-prepare → kc-convert-* → kc-finalize
```

Pipeline binaries including `kc-copy` are under `KC_BIN_DIR`
(default `/usr/lib/kc-utils`; locally `make build` → `bin/` + `KC_BIN_DIR=$PWD/bin`).

### Configuration

| Variable | Flag | Default |
|---|---|---|
| `V2V_copyConcurrency` | `--copy-concurrency` | `4` (capped at disk count; `1` = sequential) |

Also required for copy: `V2V_libvirtURL`, `V2V_fingerprint`, `V2V_vmName`,
vSphere credentials at `/etc/secret/accessKeyId` and `/etc/secret/secretKey`.
Source vmdk paths are resolved by `env.ResolveCopySources()` (govmomi inventory
or optional `V2V_diskPath` override). Disks are copied in parallel (worker pool);
the first failure cancels siblings.

### Package layout (`pkg/copy/`)

| File | Role |
|---|---|
| `copy.go` | `CopyInput`, `Run()` (parallel), progress |
| `target.go` | Discover PVC mounts, emptiness checks |
| `vsphere.go` | NFC export via govmomi (`ExportVM`, `Lease`) |
| `vmdk.go` | Stream-optimized VMDK reader (grain parsing, zlib) |
| `download.go` | Per-disk HTTP download + VMDK → raw conversion |

Pipeline stage / standalone CLI: [docs/kc-copy.md](../../docs/kc-copy.md).

---

## vsphere — vCenter inventory

Queries vCenter via govmomi (no libvirt/CGO). Replaces the libvirt
`GetXMLDesc()` path Forklift virt-v2v used for vSphere metadata and disk ordering.

Used by:

- `env.ResolveCopySources()` — ordered vmdk paths for disk copy (via `env` package)
- `env.FetchSourceMeta()` — NIC MACs, firmware, guest hostname

### Connection

| Function | File | Role |
|---|---|---|
| `connect()` | `connect.go` | `vpx://` URL → `https://host/sdk` via govmomi |
| `credentials()` | `connect.go` | User from URL or `/etc/secret/accessKeyId`; password from `/etc/secret/secretKey` |

`V2V_libvirtURL` is parsed as a libvirt-style URL (e.g.
`vpx://user@vcenter/Datacenter/no_verify=1`). The datacenter name in the path
scopes VM lookup. `no_verify=1` in the query string disables TLS verification
(matching Forklift behavior).

### Inventory

| Function | File | Role |
|---|---|---|
| `LoadInventory()` | `inventory.go` | Disks, NICs, firmware, guest ID/name/hostname |
| `disksFromDevices()` | `disks.go` | Disk ordering (SCSI → SATA → IDE → NVME) |
| `ResetCache()` | `inventory.go` | Clear process cache (tests) |

`LoadInventory()` caches results per process so copy and conversion share one
vCenter round-trip in the same pod.

Disk paths follow libvirt bus order and resolve snapshot backing chains to the
base vmdk. NIC MACs are collected from all virtual Ethernet device types
(vmxnet3, e1000, etc.).

### Required env vars

| Variable | Purpose |
|---|---|
| `V2V_libvirtURL` | vCenter connection URL |
| `V2V_vmName` | Source VM name |

Not used for EC2 or other disk-only sources.

---

## inspection/xml — Forklift inspection output

Writes the virt-v2v-style inspection XML Forklift serves at `GET /inspection`.
Used for vSphere OS mapping (`vm.OperatingSystem`).

| Function | Role |
|---|---|
| `WriteInspectionXML()` | Build `<v2v><operatingsystem>…` from `TargetMeta` |

Output path defaults to `/var/tmp/v2v/inspection.xml`
(`config.DefaultInspectionOutputFile`). Fields mapped from kc-finalize output:

| XML element | Source |
|---|---|
| `name` | `Inspect.ProductName` (fallback: `Inspect.Distro`) |
| `distro` | `Inspect.Distro` |
| `osinfo` | `Inspect.OsinfoID` (inferred from distro/version when empty) |
| `arch` | `Inspect.Arch` (default `x86_64`) |

Osinfo IDs are inferred for common Linux distros and Windows versions when
kc-finalize does not set one explicitly.
