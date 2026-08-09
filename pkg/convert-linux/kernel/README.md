# kernel -- kernel scanning, probing, and selection

Defines the `KernelScanner` interface for discovering installed kernels in a guest filesystem and selecting the best candidate for conversion. Also provides standalone functions for probing kernel module support and choosing the optimal kernel from a list.

`KernelScanner` implementations register into a plugin registry and are responsible for enumerating installed kernels (via `ScanKernels`) and picking the best one (via `SelectBest`). `ProbeModules` inspects a kernel's driver directory using a single glob to determine whether it has virtio support and whether it is a Xen PV-only kernel. `Best` filters out Xen PV-only and unbootable kernels (those without a vmlinuz path), then selects the highest-versioned kernel with a preference for virtio-capable kernels. `ModulesDir` locates the kernel modules directory, checking both `/lib/modules` and `/usr/lib/modules`.

## File layout

| File | Purpose |
|------|---------|
| `kernel.go` | `KernelScanner` interface and `Scanners` plugin registry |
| `probe.go` | `ProbeModules` for checking virtio/Xen support; `ModulesDir` for locating the modules directory |
| `select.go` | `Best` function for selecting the optimal kernel from a list |

## Key exports

| Symbol | Role |
|--------|------|
| `KernelScanner` | Interface for scanning installed kernels and selecting the best candidate (ScanKernels, SelectBest) |
| `Scanners` | Global plugin registry of `KernelScanner` implementations |
| `ProbeModules` | Checks whether a kernel version has virtio support or is Xen PV-only |
| `ModulesDir` | Returns the guest's kernel modules directory path |
| `Best` | Selects the best bootable kernel, preferring virtio-capable and highest version |
