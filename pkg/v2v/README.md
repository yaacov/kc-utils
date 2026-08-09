# kc-v2v — pkg libraries

Importable libraries for the v2v orchestrator ([`cmd/kc-v2v`](../../cmd/kc-v2v/main.go)).
Orchestration lives in [`pkg/cmd/v2v/`](../cmd/v2v/).

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
  → v2v.Run(cfg)
      → env.NeedsCopy?  → ResolveCopySources → ValidateCopySourceCount → write copy-input.json → kc-copy
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

