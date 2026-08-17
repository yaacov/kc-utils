# pkg/cmd/

Thin orchestrators that wire pipeline blocks into executable sequences. Each
orchestrator's `pipeline.go` reads a JSON input (produced by the previous
stage), runs its blocks in order, and writes a JSON output for the next stage.
The orchestrators contain no conversion logic themselves — all work is
delegated to block packages under [`pkg/`](../).

The four core binaries form a linear pipeline:
**kc-prepare** → **kc-convert-linux** or **kc-convert-windows** → **kc-finalize**.
The `kc-v2v` orchestrator wraps the full pipeline and adds vSphere integration.

| Utility | README | Orchestrator |
|---------|--------|--------------|
| **prepare** | [`prepare/README.md`](prepare/README.md) | [`prepare/pipeline.go`](prepare/pipeline.go) |
| **convert-linux** | [`convert-linux/README.md`](convert-linux/README.md) | [`convert-linux/pipeline.go`](convert-linux/pipeline.go) |
| **convert-windows** | [`convert-windows/README.md`](convert-windows/README.md) | [`convert-windows/pipeline.go`](convert-windows/pipeline.go) |
| **finalize** | [`finalize/README.md`](finalize/README.md) | [`finalize/pipeline.go`](finalize/pipeline.go) |
| **v2v** | [`v2v/README.md`](v2v/README.md) | [`v2v/pipeline.go`](v2v/pipeline.go), [`v2v/server/`](v2v/server/) |
| **agent** | (in-appliance; see [`docs/backends/kc-agent.md`](../../docs/backends/kc-agent.md)) | [`agent/pipeline.go`](agent/pipeline.go) |
| **agentsh** | (host debug shell; see [`docs/apps/kc-agent-sh.md`](../../docs/apps/kc-agent-sh.md)) | [`agentsh/run.go`](agentsh/run.go) |

Importable v2v libraries (config, env, copy, vsphere, inspection):
[`pkg/v2v/`](../v2v/README.md).

Block packages: [`pkg/`](../). User-facing docs: [`docs/`](../../docs/).
