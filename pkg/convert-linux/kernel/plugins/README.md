# kernel plugins

`KernelScanner` interface — scan installed kernels; `Best()` filters Xen-PV candidates.

| Key | Package | Description |
|-----|---------|-------------|
| `rpm` | rpm/ | Scan RPM-installed kernels |
| `deb` | deb/ | Scan DEB-installed kernels |

Kernel selection (strict) runs after all scanners register candidates.
