# Conversion Paths Reference

For identification, persistence, and stage consumption of the hypervisor, guest
OS, OS version, and network stack axes, see
[conversion-dimensions.md](conversion-dimensions.md).

This document maps every OS + source-hypervisor code path in kc-utils.
Conversion has two independent concerns:

- **Cleanup** — remove/disable drivers, services, packages, and registry
  entries left by the source hypervisor. Depends on which hypervisor the VM
  comes from.
- **Install** — install virtio drivers and guest agent so the VM boots on
  KubeVirt/KVM. Depends on the target OS.

These two concerns are tracked separately in the OS-specific references below.

**Pipeline entry points:**
- Linux: [`pkg/cmd/convert-linux/pipeline.go`](../../pkg/cmd/convert-linux/pipeline.go)
- Windows: [`pkg/cmd/convert-windows/pipeline.go`](../../pkg/cmd/convert-windows/pipeline.go)

For handler/classification detail (how guests are matched to plugins), see
[guest-os-handlers.md](guest-os-handlers.md).

## OS-specific references

- [conversion-paths-linux.md](conversion-paths-linux.md) — hypervisor cleanup and distro install matrices
- [conversion-paths-windows.md](conversion-paths-windows.md) — hypervisor cleanup and version install matrices
