# kernel plugins

`KernelScanner` interface — scan installed kernels; `Best()` filters Xen-PV candidates.

Kernel scanning discovers every installed kernel on the guest so the strict
selection step can pick the best candidate for KVM boot. Each scanner plugin
knows how to enumerate kernels for a particular package format. After all
scanners run, the strict `Best()` function scores each kernel by checking
whether its `/lib/modules/<version>/` tree contains the virtio modules needed
for early boot (virtio_blk, virtio_scsi, virtio_net, virtio_pci). Xen
paravirtualized kernels (kernel name containing `xen` or lacking virtio
modules) are deprioritized.

| Key | Package | Description |
|-----|---------|-------------|
| `rpm` | rpm/ | Scan RPM-installed kernels |
| `deb` | deb/ | Scan DEB-installed kernels |

Kernel selection (strict) runs after all scanners register candidates.

## rpm

**What it does:** Enumerates kernels installed via RPM on RHEL, CentOS, Fedora,
SUSE, and other RPM-based distributions.

**How it works:** Scans `/lib/modules/` for directories whose names match
installed kernel versions. For each discovered version, reads the module tree
to build a `KernelInfo` struct with version string, module directory path, and
a list of available modules. Handles both standard (`kernel`) and variant
(`kernel-core`, `kernel-default`) RPM naming patterns.

## deb

**What it does:** Enumerates kernels installed via dpkg on Debian and Ubuntu.

**How it works:** Scans `/lib/modules/` for installed kernel versions, similar
to the RPM scanner. Recognizes Debian-style kernel version strings
(e.g. `5.10.0-20-amd64`) and builds `KernelInfo` entries with the module
directory and available module list. Handles both `linux-image` and
`linux-modules` package layouts.
