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

- Disk copying from VMware uses pure Go
  [govmomi](https://github.com/vmware/govmomi) NFC export to stream VMDK files
  directly into target PVCs.
- Guest filesystem operations go through a pluggable backend: host-kernel
  mounts (`direct`), a [libguestfs](https://libguestfs.org/) appliance
  (`guestfs`), or our own QEMU appliance (`qemu`). Appliance backends mount
  guest disks inside a VM, so they need no host root or `CAP_SYS_ADMIN`;
  `qemu` also runs on macOS.

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

Three guest access modes are supported (see [docs/architecture/privilege-model.md](docs/architecture/privilege-model.md)):

- **host-mount** (`--backend direct`, default) - mounts guest filesystems with
  `mount(8)` and runs guest tools via `chroot` into that tree; requires root or
  `CAP_SYS_ADMIN`.
- **guestfs** (`--backend guestfs`) - runs a minimal [libguestfs](https://libguestfs.org/)
  appliance VM via `guestfish`; guest disks are accessed inside the appliance
  (no host root); requires Linux with `/dev/kvm`.
- **qemu** (`--backend qemu`) - boots our own minimal appliance directly with
  `qemu-system-*` and drives a primitive in-guest agent over a unix socket,
  composing all logic host-side. Runs on **Linux or macOS** (KVM/HVF/TCG). See
  [docs/architecture/qemu-appliance.md](docs/architecture/qemu-appliance.md).

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

- **Pure Go core pipeline** - builds with standard `go build` on any Unix host, no C toolchain
  required. Release images use `GOOS=linux` (`make build`). The `direct` and
  `guestfs` backends require Linux at runtime; the `qemu` backend also runs on
  macOS (it only launches `qemu-system-*` and speaks a unix socket).
- **Initramfs rebuild via guest tools** - virtio drivers are injected by running
  the guest's own tooling via `chroot` into the mounted guest root (host-mount)
  or an in-appliance chroot (guestfs): `dracut` first, then `update-initramfs`,
  then `mkinitramfs` as fallbacks.
- **Windows offline driver injection** - virtio drivers are registered in the
  Windows registry (`CriticalDeviceDatabase` / `DriverDatabase`) offline, making
  the guest bootable on KVM. Firstboot PowerShell scripts then complete driver
  installation via `pnputil` and install the QEMU guest agent.
- **ARM / aarch64 support** - cross-architecture conversion works out of the box.
  `kc-copy` uses pure Go (govmomi NFC), so it compiles and runs on any Unix host.
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
| `kc-guest-agent` | In-appliance PID 1 for `--backend qemu` (not run on the conversion host) |

The tree compiles on any Unix host. Guest disk backends (`direct`, `guestfs`) require
Linux at runtime; `qemu` also runs on macOS. `kc-copy` runs on any Unix. Release
binaries are built with `make build` (`GOOS=linux`).

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
- [docs/apps/kc-guest-agent.md](docs/apps/kc-guest-agent.md) - in-appliance agent for the qemu backend
- [pkg/v2v/README.md](pkg/v2v/README.md) - kc-v2v libraries (copy, vsphere, env, inspection)
- [build/kc-v2v/README.md](build/kc-v2v/README.md) - Container image, Forklift Plan config
- [docs/apps/forklift-usage.md](docs/apps/forklift-usage.md) - Using kc-v2v with Forklift (MTV)
- [docs/apps/examples/](docs/apps/examples/README.md) - JSON samples and runnable example
- [docs/apps/kc-prepare.md](docs/apps/kc-prepare.md) - kc-prepare pipeline
- [docs/apps/kc-convert-linux.md](docs/apps/kc-convert-linux.md) - Linux converter pipeline
- [docs/apps/kc-convert-windows.md](docs/apps/kc-convert-windows.md) - Windows converter pipeline
- [docs/apps/kc-finalize.md](docs/apps/kc-finalize.md) - kc-finalize pipeline

### Debug

- [docs/debug/README.md](docs/debug/README.md) - local Mac/Linux qemu conversion cookbook

### Architecture

- [docs/architecture/README.md](docs/architecture/README.md) - Architecture reference index
- [docs/architecture/privilege-model.md](docs/architecture/privilege-model.md) - Privilege model: host-mount vs guestfish / libguestfs appliance
- [docs/architecture/qemu-appliance.md](docs/architecture/qemu-appliance.md) - qemu backend: our own minimal appliance, primitive agent protocol, host/guest logic split
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
