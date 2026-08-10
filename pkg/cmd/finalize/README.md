# kc-finalize — orchestrator

[`pipeline.go`](pipeline.go) is a thin orchestrator for kc-finalize. It runs 7
semantic blocks in order under [`pkg/finalize/`](../../finalize/).

**Pluggable:** `pkg/finalize/<block>/plugins/` (customize).

**Strict:** `pkg/finalize/<block>/` (target, metadata). Blocks 2–5 are inline `pkg/guest` calls.

Writes `TargetMeta` JSON on success.

| Block | Package | Type |
|-------|---------|------|
| 1 customize | [`pkg/finalize/customize/`](../../finalize/customize/) | pluggable |
| 2 fstrim | inline (`pkg/guest/`) | inline |
| 3 unmount | inline (`pkg/guest/`) | inline |
| 4 post-fsck | inline (`pkg/guest/`) | inline |
| 5 release | inline (`pkg/guest/`) | inline |
| 6 target | [`pkg/finalize/target/`](../../finalize/target/) | strict |
| 7 metadata | [`pkg/finalize/metadata/`](../../finalize/metadata/) | strict |

Full block table: [`docs/apps/kc-finalize.md`](../../../docs/apps/kc-finalize.md).
