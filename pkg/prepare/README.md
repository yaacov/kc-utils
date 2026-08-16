# kc-prepare blocks

All pipeline blocks for [`cmd/kc-prepare`](../../cmd/kc-prepare/main.go). Pluggable
blocks register implementers under `plugins/` (see each block's `plugins/README.md`).
Each block has its own README with detailed exports and mechanism.

**Type:** `strict` = single built-in implementation · `pluggable` = implementation chosen from a `plugins/` registry ([plugin model](../../community/architecture.md#plugin-system)) · `strict + pluggable` = registry plus built-in wiring/fallback · `inline` = handled directly by the stage orchestrator.

| # | Block | Package | Type | Description |
|---|-------|---------|------|-------------|
| 1 | Validate | [`validate/`](validate/) | strict | Validate disk list and create mount root |
| 2 | Guest | [`guest/`](guest/) | strict | Open disks, scan partitions, activate LVM |
| 3 | Decrypt | inline (`pkg/guest/`) | inline | LUKS decryption via `Guest.Decrypt`/`UnlockClevis` (after open/LVM; `all` tries every device) |
| 4 | Pre-Fsck | inline (`pkg/guest/`) | inline | Pre-conversion fsck |
| 5 | Firmware | [`firmware/`](firmware/) | pluggable | BIOS vs UEFI detection (also refreshed after mount) |
| 6 | Root | [`root/`](root/) | strict + pluggable | Root discovery and selection (default `first`) |
| 7 | Mount | [`mount/`](mount/) | strict + pluggable | Mount planning and execution |
| 8 | Inspect | [`inspect/`](inspect/) | strict | OS inspection, boot device, free space |
| 9 | Converter | [`converter/`](converter/) | pluggable | Choose linux/windows converter |

Orchestrator: [`pkg/cmd/prepare/`](../cmd/prepare/).
Docs: [`docs/apps/kc-prepare.md`](../../docs/apps/kc-prepare.md).
