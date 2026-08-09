# kc-finalize blocks

All pipeline blocks for [`cmd/kc-finalize`](../../cmd/kc-finalize/main.go). Pluggable
blocks document implementers in `<block>/plugins/README.md`. Each block has its
own README with detailed exports and mechanism.

| # | Block | Package | Type | Description |
|---|-------|---------|------|-------------|
| 1 | Customize | [`customize/`](customize/) | pluggable | Post-mount customization |
| 2 | Fstrim | [`fstrim/`](fstrim/) | pluggable | Run `fstrim` on mounted filesystems |
| 3 | Teardown | inline (`pkg/guest/`) | inline | Unmount, deactivate LVM, close LUKS |
| 4 | Post-Fsck | inline (`pkg/guest/`) | inline | Post-conversion filesystem check |
| 5 | Target | [`target/`](target/) | strict | Firmware resolution, disk/NIC bus slots |
| 6 | Metadata | [`metadata/`](metadata/) | strict | Assemble and write TargetMeta JSON |

Orchestrator: [`pkg/cmd/finalize/`](../cmd/finalize/).
Docs: [`docs/apps/kc-finalize.md`](../../docs/apps/kc-finalize.md).
