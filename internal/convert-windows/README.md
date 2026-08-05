# kc-convert-windows — orchestrator

[`pipeline.go`](pipeline.go) is a thin orchestrator for kc-convert-windows. It
runs 9 semantic blocks in order under [`pkg/convert-windows/`](../../pkg/convert-windows/).

**Pluggable:** `pkg/convert-windows/<block>/plugins/` (driversource, hypervisor, drivers).

**Strict:** `pkg/convert-windows/<block>/` (inspect, crashcontrol, firstboot, ntfsfix,
output).

Writes `ConverterOutput` JSON on success.

Full block table: [`docs/kc-convert-windows.md`](../../docs/kc-convert-windows.md).
