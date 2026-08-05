# kc-prepare — orchestrator

[`pipeline.go`](pipeline.go) is a thin orchestrator for kc-prepare. It calls
strict blocks under [`pkg/prepare/`](../../pkg/prepare/) and looks up pluggable blocks
from each block's `plugins/` registry.

**Sequence:** validate → guest open → decrypt → fsck → root discovery/selection
→ mount plan/execute → inspect → firmware → converter selection.

Writes `PrepareOutput` JSON on success; on multiboot failure writes partial output with
`root_candidates`.

| Block | Package | Type | Description |
|-------|---------|------|-------------|
| 1 | [`pkg/prepare/validate/`](../../pkg/prepare/validate/) | strict | Validate input disks and create mount root |
| 2 | [`pkg/prepare/guest/`](../../pkg/prepare/guest/) | strict | Open disks, scan partitions, activate LVM |

Guest sub-packages include disk I/O, LUKS/LVM, overlay (kc-v2v), and device
resolution — see [`pkg/prepare/README.md`](../../pkg/prepare/README.md#guest-sub-packages).

| 3 | inline (`pkg/guest/`) | inline | LUKS decryption |
| 4 | inline (`pkg/guest/`) | inline | Pre-conversion fsck |
| 5 | [`pkg/prepare/root/`](../../pkg/prepare/root/) | strict + pluggable | Root discovery and selection |
| 6 | [`pkg/prepare/mount/`](../../pkg/prepare/mount/) | strict + pluggable | Mount planning and execution |
| 7 | [`pkg/prepare/inspect/`](../../pkg/prepare/inspect/) | strict | OS inspection, boot device, free space |
| 8 | [`pkg/prepare/firmware/`](../../pkg/prepare/firmware/) | pluggable | Firmware detection |
| 9 | [`pkg/prepare/converter/`](../../pkg/prepare/converter/) | pluggable | Converter selection |

Full block table: [`docs/kc-prepare.md`](../../docs/kc-prepare.md).
