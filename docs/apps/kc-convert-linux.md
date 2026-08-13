# kc-convert-linux Pipeline

Converts a Linux guest to run on KVM/virtio. Removes source hypervisor tools,
injects virtio drivers into the initramfs, remaps device names, and fixes
the bootloader configuration.

Requires Linux (`//go:build linux`).

## Entry Point

`cmd/kc-convert-linux/main.go` — orchestrator in `pkg/cmd/convert-linux/pipeline.go`.

## CLI Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--input` | yes | | Path to PipelineData JSON (from kc-prepare) |
| `--output` | no | `convert-out.json` | Path to write PipelineData JSON (with `convert` section added) |
| `--mount-root` | no | `/tmp/kc-guest` | Host directory where guest filesystems are mounted |
| `--offline` | no | `false` | Skip network firstboot when no local package matches (local packages are always tried first) |
| `--guestfs` | no | `false` | Use libguestfs appliance instead of privileged mount syscalls |
| `--log-level` | no | `info` | Log level (`debug`, `info`, `warn`, `error`) |

## qemu-guest-agent Installation

The converter always searches for a matching local package first, regardless
of the `--offline` flag. If `/usr/bin/qemu-ga` is already present, conversion
skips package installation only when `qemu-guest-agent.service` is enabled and
not masked. A pre-installed but disabled or masked unit is enabled offline
during conversion (admin wants symlink under `/etc/systemd/system/`), with
firstboot `systemctl unmask` / `systemctl enable --now` as fallback when the
unit file is missing. The decision flow is:

1. **Binary present and unit operational** -- skip guest-agent work entirely.
2. **Binary present but unit disabled/masked** -- enable offline, or schedule
   enable-only firstboot when the unit file is missing.
3. **Local package found** -- copy it into the guest (`/var/lib/kc-packages/`)
   and install at firstboot via `rpm -ivh` or `dpkg -i`. No network required.
4. **No local package found, `--offline=false`** (default) -- add a firstboot
   script that installs via `dnf`/`yum`/`apt`/`zypper` after network is up.
5. **No local package found, `--offline=true`** -- skip guest-agent
   installation entirely (no firstboot script added).

`kc-v2v` passes `--offline` when `V2V_offline=true` so convertor pods use
image-staged packages without falling back to network install.

### Host package layout (RHEL family)

Default base: `/usr/share/kc-packages`

```text
/usr/share/kc-packages/rpm/el8/x86_64/qemu-guest-agent-*.rpm
/usr/share/kc-packages/rpm/el9/x86_64/qemu-guest-agent-*.rpm
/usr/share/kc-packages/rpm/el10/x86_64/qemu-guest-agent-*.rpm
```

Selection uses inspect `distro` + `major_version`: prefer exact `el{major}`, else nearest lower `elN` that exists. Only **one** RPM is copied into the guest.

The kc-v2v image stages el8/el9/el10 x86_64 RPMs via [`build/kc-v2v/stage-linux-packages.sh`](../build/kc-v2v/stage-linux-packages.sh). Legacy flat layout `$base/rpm/$arch/` remains supported for one-off mounts.

Debian/SUSE guests are not covered by the baked matrix; without a matching local package, `--offline` skips agent install.

## Pipeline Blocks

| # | Block | Type | Package | Description |
|---|-------|------|---------|-------------|
| 1 | Distro | pluggable: `DistroHandler` | `pkg/convert-linux/distro/` | Classify OS family |
| 2 | Package Format | strict | `pkg/convert-linux/distro/` | Determine package format (rpm/deb) |
| 3 | Package Manager | strict | `pkg/convert-linux/distro/` | Determine package manager name |
| 4 | Bootloader | pluggable: `BootloaderHandler` | `pkg/convert-linux/bootloader/` | Detect boot config format (grub2, bls) |
| 5 | Kernel Scan | pluggable: `KernelScanner` | `pkg/convert-linux/kernel/` | Scan installed kernels |
| 6 | Remap | pluggable: `DeviceRemapper` | `pkg/convert-linux/remap/` | Rewrite block device names in `/etc/fstab`, `/etc/crypttab`, and bootloader kernel args |
| 7 | UEFI | pluggable: `UEFIEditor` | `pkg/common/uefi/` | Update UEFI boot entries on ESP partitions |
| 8 | Kernel Select | strict | `pkg/convert-linux/kernel/` | Select best virtio-capable kernel |
| 9 | Console Config | strict | `pkg/convert-linux/bootconfig/` | Configure serial console kernel args |
| 10 | Display Config | strict | `pkg/convert-linux/bootconfig/` | Configure virtio video kernel args |
| 11 | Hypervisor | pluggable: `LinuxCleanup` | `pkg/convert-linux/hypervisor/` | Remove source hypervisor tools; EC2 masks IMDS net hooks |
| 11b | Network | `network.Select` | `pkg/convert-linux/network/` | Exclusive handler: networkd virtio DHCP + wait-online, or no-op (default) |
| 12 | Guest Agent | pluggable | `pkg/convert-linux/guestagent/` | Install qemu-guest-agent, local or firstboot network packages |
| 13 | Guest Cleanup | strict | `pkg/convert-linux/guestcleanup/` | Remove blkid/LVM caches and update modprobe aliases |
| 14 | Initramfs | strict | `pkg/convert-linux/initramfs/` | Rebuild initramfs with virtio drivers |
| 15 | Static IP / NIC Naming | `network.Select` + `NICNamer` | `pkg/convert-linux/network/`, `pkg/convert-linux/nicnaming/` | networkd handler: MAC `.network` files; default handler: nicnaming + staticip firstboot |
| 16 | SELinux Relabel | strict | `pkg/convert-linux/selinux/` | Offline SELinux relabel via `setfiles` |
| 17 | GuestCaps | strict | `pkg/convert-linux/guestcaps/` | Derive guest capabilities for KVM |

Block numbers match the pipeline comments in `pkg/cmd/convert-linux/pipeline.go`.

Distro handlers (`rhel`, `debian`, `suse`), package format/manager lookup, and
offline QGA selection are documented in
[guest-os-handlers.md](guest-os-handlers.md).

## Input

- `PipelineData` JSON (from kc-prepare): OS info, disk layout, mount paths (in `prepare` section)
- Mounted guest filesystem at `--mount-root`

Example: [examples/prepare-output-complete.json](examples/prepare-output-complete.json).

## Output

`ConverterOutput` JSON containing:

- `guestcaps` -- guest capabilities derived during conversion:
  - `block_bus` -- `virtio` or `ide` (fallback when no virtio modules found)
  - `net_bus` -- `virtio` or `e1000`
  - virtio feature flags (RNG, balloon, socket, etc.)
  - `machine_type` -- `q35` (x86_64) or `virt` (aarch64)
- `hypervisor` -- in-guest hypervisor plugin outcomes (only when at least one plugin matched):
  - `plugins[].name` -- registry key (for example `vmware`, `ec2`)
  - `plugins[].action` -- `cleanup`
  - `plugins[].status` -- `succeeded` or `failed`
  - `plugins[].error` -- present when status is `failed`
- `network` -- selected network handler (Linux only):
  - `handler` -- `networkd` or `default`
  - `primary` -- `systemd-networkd` or `legacy`

Example: [examples/convert-output-linux.json](examples/convert-output-linux.json).

## Plugin Implementations

| Interface | Implementations |
|-----------|----------------|
| `DistroHandler` | `rhel`, `debian`, `suse`, ... |
| `BootloaderHandler` | `grub2`, `bls` |
| `KernelScanner` | `rpm`, `deb` |
| `DeviceRemapper` | `standard` |
| `LinuxCleanup` | `vmware`, `virtualbox`, `citrix`, `parallels`, `xen`, `kudzu`, `hyperv`, `ec2`, `nutanix`, ... |
| `GuestAgent` | `qemu-ga` |
| `PackageSource` | `directory` (local RPM/DEB packages) |
| `FirstbootHandler` | `systemd` |
| `NICNamer` | `nm`, `netplan`, `ifcfg`, `wicked`, `dhclient`, `nmdhcp` |
| `NetworkHandler` | `networkd`, `default` |
| `UEFIEditor` | `grub-fallback` |

## Initramfs Rebuild Strategy

The converter rebuilds the guest initramfs using the guest's own tooling
via `RunInGuest`:

1. Back up the existing initramfs (`{path}.pre-v2v`)
2. Try `dracut --force --add-drivers "virtio ..." {path} {version}` (RHEL/Fedora/SUSE)
3. Fall back to `update-initramfs -u -k {version}` (Debian/Ubuntu)
4. Fall back to `mkinitramfs -o {path} {version}` (older Debian)

This approach uses the guest's native module compression, dependency
resolution, and firmware inclusion rather than custom CPIO manipulation.

## See also

- [../architecture/conversion-paths-linux.md](../architecture/conversion-paths-linux.md) — Linux hypervisor cleanup and distro install matrices
- [../architecture/guest-os-handlers.md](../architecture/guest-os-handlers.md) — distro handler classification
