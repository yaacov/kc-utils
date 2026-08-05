# kc-finalize — orchestrator

[`pipeline.go`](pipeline.go) is a thin orchestrator for kc-finalize. It runs 6
semantic blocks in order under [`pkg/finalize/`](../../pkg/finalize/).

**Pluggable:** `pkg/finalize/<block>/plugins/` (customize).

**Strict:** `pkg/finalize/<block>/` (target, metadata). Blocks 2-4 are inline `pkg/guest` calls.

Writes `TargetMeta` JSON on success.

| Block | Package | Type |
|-------|---------|------|
| 1 customize | [`pkg/finalize/customize/`](../../pkg/finalize/customize/) | pluggable |
| 2 fstrim | inline (`pkg/guest/`) | inline |
| 3 teardown | inline (`pkg/guest/`) | inline |
| 4 fschecker | inline (`pkg/guest/`) | inline |
| 5 target | [`pkg/finalize/target/`](../../pkg/finalize/target/) | strict |
| 6 metadata | [`pkg/finalize/metadata/`](../../pkg/finalize/metadata/) | strict |

Full block table: [`docs/kc-finalize.md`](../../docs/kc-finalize.md).
