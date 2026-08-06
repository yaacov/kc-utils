# kc-finalize blocks

All pipeline blocks for [`cmd/kc-finalize`](../../cmd/kc-finalize/main.go). Pluggable
blocks document implementers in `<block>/plugins/README.md`.

| # | Block | Package | Type | Description |
|---|-------|---------|------|-------------|
| 1 | Customize | [`customize/`](customize/) | pluggable | Post-mount customization |
| 2 | Fstrim | [`fstrim/`](fstrim/) | inline | Run `fstrim` on mounted filesystems |
| 3 | Teardown | inline (`pkg/guest/`) | inline | Unmount, deactivate LVM, close LUKS |
| 4 | Post-Fsck | inline (`pkg/guest/`) | inline | Post-conversion filesystem check |
| 5 | Target | [`target/`](target/) | strict | Firmware resolution, disk/NIC bus slots |
| 6 | Metadata | [`metadata/`](metadata/) | strict | Assemble and write TargetMeta JSON |

## Customize sub-packages

[`customize/`](customize/) runs post-mount customization scripts and file injection.
Plugins: [`customize/plugins/README.md`](customize/plugins/README.md).

| Sub-package | Role |
|---|---|
| [`plugins/native/`](customize/plugins/native/) | Direct file injection without external binary |
| [`plugins/dynamicscripts/`](customize/plugins/dynamicscripts/) | Run external customization scripts |

## Target sub-packages

[`target/`](target/) resolves firmware and assigns KubeVirt bus slots:

| File | Role |
|---|---|
| [`fwresolve.go`](target/fwresolve.go) | BIOS vs UEFI from prepare firmware + guest layout |
| [`busassign.go`](target/busassign.go) | Disk and NIC bus assignment for TargetMeta |

Orchestrator: [`pkg/cmd/finalize/`](../cmd/finalize/).
Docs: [`docs/kc-finalize.md`](../../docs/kc-finalize.md).
