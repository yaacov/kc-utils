# Kvm Converter Utilities (kc-utils)

Pure Go utilities that convert virtual machines from external hypervisors into
KVM-compatible guests. Guest adaptations include VirtIO driver injection,
initramfs and kernel updates, boot configuration fixes (device remapping, kernel
args, UEFI), and network migration (VirtIO net drivers, NIC naming, static IPs)
for Linux and Windows.

The core pipeline is four pure-Go binaries that inspect and mount guest disks,
convert the guest OS, then unmount, validate filesystems, and emit JSON metadata
for creating the target VM.

<p align="center">
  <br>
  <img src="docs/logo.png" alt="kc-utils logo" width="420">
  <br><br>
</p>

kc-utils are a pure Go re-implementation of [virt-v2v-in-place](https://libguestfs.org/virt-v2v-in-place.1.html);
see also [virt-v2v](https://libguestfs.org/virt-v2v.1.html).

- Disk copying from VMware uses pure Go
  [govmomi](https://github.com/vmware/govmomi) NFC export to stream VMDK files
  directly into target PVCs (no VDDK or nbdkit required).
- Guest filesystem operations can run rootless via the
  [libguestfs](https://libguestfs.org/) appliance, driven by the `guestfish`
  CLI: a minimal QEMU appliance VM mounts the guest disks internally, so no
  host root or `CAP_SYS_ADMIN` is required.

**[Benchmark](docs/architecture/ref-baseline/README.md)** :
On OpenShift MTV cold migrations, kc-v2v beats virt-v2v wall time
(about 3.5 minutes faster on RHEL, about 6 minutes faster on Windows) while
using less peak memory and far less peak CPU, and transferring roughly
57 % / 38 % less network data (RHEL / Windows) by converting disks in-pod
instead of a separate DiskTransferV2v stream.

**[Dashboard](https://htmlpreview.github.io/?https://github.com/yaacov/kc-utils/blob/main/docs/architecture/ref-baseline/dashboard.html)**
([source](docs/architecture/ref-baseline/dashboard.html)):
Interactive charts of memory, CPU, and network I/O over time for the ref vs
kc-v2v runs.

Two guest access modes plus a QEMU appliance backend are supported (see [docs/backends/README.md](docs/backends/README.md)). `--backend` / `V2V_backend` is required (no default):

- **host-mount** (`--backend=direct`) - mounts guest filesystems with
  `mount(8)` and runs guest tools via `chroot` into that tree; requires Linux
  root or `CAP_SYS_ADMIN`. Registered on Linux only.
- **guestfs** (`--backend=guestfs` / `V2V_backend=guestfs`) - runs a minimal
  [libguestfs](https://libguestfs.org/) appliance VM via `guestfish`; guest disks
  are accessed inside the appliance (no host root); requires Linux with
  `/dev/kvm`. The `kc-v2v` container image sets `V2V_backend=guestfs` explicitly.
- **qemu** (`--backend=qemu`) - boots a shipped kernel+initramfs under QEMU and
  talks to `kc-agent` over a virtio-serial Unix socket. Host needs QEMU plus
  appliance files (`vmlinuz`, `initramfs.img`); conversion
  tools run inside the appliance. Registered on Linux and macOS.

## Forklift (MTV) Integration

kc-v2v is a drop-in replacement for the virt-v2v **container image** in
[Forklift](https://github.com/kubev2v/forklift). The MTV cluster setting is still
`virt_v2v_image_fqin`; point it at your kc-v2v image FQIN:

```bash
oc mtv settings set --setting virt_v2v_image_fqin \
  --value quay.io/yaacov/kc-v2v:devel-amd64
```

See [docs/apps/forklift-usage.md](docs/apps/forklift-usage.md) for full usage instructions.

## Design Highlights

- **Pure Go core pipeline** - builds with standard `go build`, no C toolchain
  required. Stage binaries target Unix (`GOOS=linux` or `GOOS=darwin`). `kc-agent`
  is Linux-only (QEMU appliance pid 1).
- **Initramfs rebuild via guest tools** - virtio drivers are injected by running
  the guest's own tooling via `chroot` into the mounted guest root (host-mount)
  or an in-appliance chroot (guestfs): `dracut` first, then `update-initramfs`,
  then `mkinitramfs` as fallbacks.
- **Windows offline driver injection** - virtio drivers are registered in the
  Windows registry (`CriticalDeviceDatabase` / `DriverDatabase`) offline, making
  the guest bootable on KVM. Firstboot PowerShell scripts then complete driver
  installation via `pnputil` and install the QEMU guest agent.
- **ARM / aarch64 support** - cross-architecture conversion works out of the box.
  `kc-copy` uses pure Go (govmomi NFC), so it runs on any architecture.
- **Pluggable architecture** - Go interfaces with a generic `Registry[K,V]` and
  `init()` self-registration. Add a new hypervisor or distro by dropping a file
  into `pkg/<utility>/<block>/plugins/`.

## Architecture

The core pipeline is four binaries executed in sequence by an external
orchestrator (`kc-v2v`, a shell script, etc.):

| Binary | Purpose |
|----|-----|
| `kc-prepare` | Open disks, inspect guest OS, mount filesystems, collect metadata |
| `kc-convert-linux` | Convert Linux guests: remove hypervisor tools, inject virtio, fix bootloader |
| `kc-convert-windows` | Convert Windows guests: install virtio-win drivers, update registry, firstboot scripts |
| `kc-finalize` | Unmount, trim, fsck, assign bus slots, determine firmware, emit TargetMeta JSON |
| `kc-v2v` | V2V orchestrator for Forklift: runs the pipeline + inspection HTTP (optional NFC disk copy for blank PVCs) |
| `kc-copy` | NFC disk copy stage via govmomi (spawned by `kc-v2v`; also usable standalone) |

Stage binaries are Unix (`//go:build unix` / `GOOS=linux` or `GOOS=darwin`).
`kc-agent` is Linux-only (QEMU appliance pid 1). `direct` and `guestfs` register
at runtime on Linux only; Darwin lists `--backend=qemu`.

Inter-app communication uses JSON files written to a shared directory, plus a
shared mount point where the guest root filesystem is mounted.

## Develop

See [community/CONTRIBUTING.md](community/CONTRIBUTING.md) for directory layout,
dependencies, build, test, and PR guidance.

## Documentation

### Apps

- [docs/README.md](docs/README.md) - Documentation index
- [docs/apps/README.md](docs/apps/README.md) - Complete conversion flow
- [docs/apps/kc-v2v.md](docs/apps/kc-v2v.md) - V2V orchestrator (Forklift conversion pod)
- [docs/apps/kc-copy.md](docs/apps/kc-copy.md) - NFC disk copy stage CLI
- [pkg/v2v/README.md](pkg/v2v/README.md) - kc-v2v libraries (copy, vsphere, env, inspection)
- [build/kc-v2v/README.md](build/kc-v2v/README.md) - Container image, Forklift Plan config
- [docs/apps/forklift-usage.md](docs/apps/forklift-usage.md) - Using kc-v2v with Forklift (MTV)
- [docs/apps/examples/](docs/apps/examples/README.md) - JSON samples and runnable example
- [docs/apps/kc-prepare.md](docs/apps/kc-prepare.md) - kc-prepare pipeline
- [docs/apps/kc-convert-linux.md](docs/apps/kc-convert-linux.md) - Linux converter pipeline
- [docs/apps/kc-convert-windows.md](docs/apps/kc-convert-windows.md) - Windows converter pipeline
- [docs/apps/kc-finalize.md](docs/apps/kc-finalize.md) - kc-finalize pipeline

### Backends

- [docs/backends/README.md](docs/backends/README.md) - Guest disk backends (direct / guestfs / qemu): selection and comparison
- [docs/backends/appliance.md](docs/backends/appliance.md) - QEMU appliance artifacts and build
- [docs/backends/kc-agent.md](docs/backends/kc-agent.md) - In-appliance RPC agent

### Architecture

- [docs/architecture/README.md](docs/architecture/README.md) - Architecture reference index
- [docs/architecture/guest-os-handlers.md](docs/architecture/guest-os-handlers.md) - Linux distro and Windows version classification, special cases, and code map
- [docs/architecture/conversion-paths.md](docs/architecture/conversion-paths.md) - OS + source-hypervisor conversion path reference

### Contributing

- [community/architecture.md](community/architecture.md) - Design principles for contributors and agents
- [community/CONTRIBUTING.md](community/CONTRIBUTING.md) - Build, test, layout, and dependencies
- [community/commits.md](community/commits.md) - Commit subject and message body conventions
- [community/pull-requests.md](community/pull-requests.md) - Branch naming and PR writing guidelines
- [community/code-review.md](community/code-review.md) - Code review priorities and report shape

## License

This project is licensed under the [GNU General Public License v3.0](LICENSE).
