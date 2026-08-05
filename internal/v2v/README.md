# kc-v2v — orchestrator

[`pipeline.go`](pipeline.go) is the kc-v2v orchestrator. It wires libraries under
[`pkg/v2v/`](../../pkg/v2v/) and spawns pipeline binaries as subprocesses.

**Sequence:** optional `kc-copy` → discover disks → kc-prepare → kc-convert-* →
kc-finalize → inspection XML → HTTP server (when `LOCAL_MIGRATION=true`).

| Step | Package | Description |
|------|---------|-------------|
| Bootstrap | [`cmd/kc-v2v`](../../cmd/kc-v2v/main.go) | `env.Load`, link certs, create workdir |
| Copy | `kc-copy` ([`pkg/copy/`](../../pkg/copy/)) | Optional NFC disk copy subprocess |
| Discover | [`pkg/v2v/env/disks.go`](../../pkg/v2v/env/disks.go) | Glob `/dev/blockN` and `/mnt/disks/...` |
| Metadata | [`pkg/v2v/env/sourcemeta.go`](../../pkg/v2v/env/sourcemeta.go) | NICs, firmware, hostname (govmomi for vSphere) |
| Pipeline | [`pipeline.go`](pipeline.go) | Spawn kc-copy → kc-prepare → converter → kc-finalize |
| Inspection | [`pkg/v2v/inspection/xml/`](../../pkg/v2v/inspection/xml/) | Write Forklift-compatible inspection XML |
| HTTP | [`server/`](server/) | Serve `/vm`, `/inspection`, `/warnings`, `/shutdown` on `:8080` |

When `V2V_overlayEnabled=true` (default), the pipeline wraps conversion in a
qcow2 overlay so block devices are not modified until commit.

Full flow and configuration: [`docs/kc-v2v.md`](../../docs/kc-v2v.md).
Library details: [`pkg/v2v/README.md`](../../pkg/v2v/README.md).
