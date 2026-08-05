# internal/

Thin orchestrators per utility. All pipeline blocks live under [`pkg/`](../pkg/).

| Utility | README | Orchestrator |
|---------|--------|--------------|
| **prepare** | [`prepare/README.md`](prepare/README.md) | [`prepare/pipeline.go`](prepare/pipeline.go) |
| **convert-linux** | [`convert-linux/README.md`](convert-linux/README.md) | [`convert-linux/pipeline.go`](convert-linux/pipeline.go) |
| **convert-windows** | [`convert-windows/README.md`](convert-windows/README.md) | [`convert-windows/pipeline.go`](convert-windows/pipeline.go) |
| **finalize** | [`finalize/README.md`](finalize/README.md) | [`finalize/pipeline.go`](finalize/pipeline.go) |
| **v2v** | [`v2v/README.md`](v2v/README.md) | [`v2v/pipeline.go`](v2v/pipeline.go), [`v2v/server/`](v2v/server/) |

Importable v2v libraries (config, env, copy, vsphere, inspection):
[`pkg/v2v/`](../pkg/v2v/README.md).

Block packages: [`pkg/`](../pkg/). User-facing docs: [`docs/`](../docs/).
