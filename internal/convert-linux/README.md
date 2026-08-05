# kc-convert-linux — orchestrator

[`pipeline.go`](pipeline.go) is a thin orchestrator for kc-convert-linux. It
runs 16 semantic blocks in order under [`pkg/convert-linux/`](../../pkg/convert-linux/).

**Pluggable:** `pkg/convert-linux/<block>/plugins/` (distro, bootloader, remap, kernel,
uefi, hypervisor, guestagent, nicnaming).

**Strict:** `pkg/convert-linux/<block>/` (bootconfig, guestcleanup, initramfs, guestcaps).

Writes `ConverterOutput` JSON on success.

Full block table: [`docs/kc-convert-linux.md`](../../docs/kc-convert-linux.md).
