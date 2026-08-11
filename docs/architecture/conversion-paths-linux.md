# Linux Conversion Paths

Maps every Linux source-hypervisor cleanup path and distro install path in kc-utils.

## Linux: Cleanup (source hypervisor removal)

Every Linux hypervisor cleanup plugin runs independently of the distro.
Each plugin detects its own artifacts and cleans them up.

**Plugin interface:** [`pkg/convert-linux/hypervisor/hypervisor.go`](../../pkg/convert-linux/hypervisor/hypervisor.go)
**Plugin directory:** [`pkg/convert-linux/hypervisor/plugins/`](../../pkg/convert-linux/hypervisor/plugins/)

### VMware

**Code:** [`pkg/convert-linux/hypervisor/plugins/vmware/cleanup.go`](../../pkg/convert-linux/hypervisor/plugins/vmware/cleanup.go)

| Action | Details |
|--------|---------|
| Services disabled | `vmtoolsd.service`, `open-vm-tools.service`, `vgauthd.service` |
| Repos disabled | All yum `.repo` files containing `vmware.com` set to `enabled=0` |
| Directories removed | `/etc/vmware-tools`, `/usr/lib/vmware-tools`, `/usr/lib64/vmware-tools` |
| Packages removed (firstboot) | `open-vm-tools`, `open-vm-tools-desktop`, `VMwareTools` via oneshot systemd unit |

### Hyper-V

**Code:** [`pkg/convert-linux/hypervisor/plugins/hyperv/cleanup.go`](../../pkg/convert-linux/hypervisor/plugins/hyperv/cleanup.go)

| Action | Details |
|--------|---------|
| Services disabled | `hv-kvp-daemon.service`, `hv-fcopy-daemon.service`, `hv-vss-daemon.service`, `hypervkvpd.service`, `hypervfcopyd.service`, `hypervvssd.service` |
| Files removed | None |
| Binaries left in place | Integration service daemons under `/usr/sbin/hv_*` are not removed offline |
| Packages removed | None |

### Citrix/XenServer

**Code:** [`pkg/convert-linux/hypervisor/plugins/citrix/cleanup.go`](../../pkg/convert-linux/hypervisor/plugins/citrix/cleanup.go)

| Action | Details |
|--------|---------|
| Services disabled | `xe-daemon.service`, `xapi.service`, `xe-linux-distribution.service` |
| Files removed | `/etc/xensource-inventory`, `/usr/sbin/xe-daemon` |
| Extra | Restores commented-out getty lines in `/etc/inittab` |

### AWS EC2

**Code:** [`pkg/convert-linux/hypervisor/plugins/ec2/cleanup.go`](../../pkg/convert-linux/hypervisor/plugins/ec2/cleanup.go)

| Action | Details |
|--------|---------|
| Services disabled and masked | `amazon-ssm-agent.service`, `amazon-cloudwatch-agent.service`, `ec2-instance-connect.service`, `hibagent.service`, `hibinit-agent.service` |
| Cloud-init | Patches `cloud.cfg` and `cloud.cfg.d/*.cfg` containing `Ec2` to `datasource_list: [None]`; writes `99-kc-disable-ec2.cfg` as fallback |
| EC2 net hooks masked | `set-hostname-imds.service`, `policy-routes@*`, `refresh-policy-routes@*` via [`systemd.DisableEC2NetHooks`](../../pkg/convert-linux/systemd/systemd.go) |
| Binaries left in place | Agent binaries/config (e.g. `/usr/bin/amazon-ssm-agent`) are not removed offline |

### Nutanix AHV

**Code:** [`pkg/convert-linux/hypervisor/plugins/nutanix/cleanup.go`](../../pkg/convert-linux/hypervisor/plugins/nutanix/cleanup.go)

| Action | Details |
|--------|---------|
| Services disabled and masked | `ngt_guest_agent.service`, `ngt_self_service_restore.service`, `nutanix-guest-agent.service` |
| Files removed | `/etc/rc.d/init.d/ngt_guest_agent`, `/etc/init.d/ngt_guest_agent`, `/usr/local/nutanix/ngt` |

### Parallels

**Code:** [`pkg/convert-linux/hypervisor/plugins/parallels/cleanup.go`](../../pkg/convert-linux/hypervisor/plugins/parallels/cleanup.go)

| Action | Details |
|--------|---------|
| Services disabled | `prltoolsd.service`, `prl-xorg-cleanup.service`, `prl-x11.service` |
| Directories removed | `/usr/lib/parallels-tools`, `/usr/lib64/parallels-tools` |

### VirtualBox

**Code:** [`pkg/convert-linux/hypervisor/plugins/virtualbox/cleanup.go`](../../pkg/convert-linux/hypervisor/plugins/virtualbox/cleanup.go)

| Action | Details |
|--------|---------|
| Services disabled | `vboxadd-service.service`, `vboxadd.service`, `vboxservice.service` |
| Directories removed | Custom install dir (from config), `/var/lib/VBoxGuestAdditions`, `/opt/VBoxGuestAdditions` |

### Xen (kernel modules)

**Code:** [`pkg/convert-linux/hypervisor/plugins/xen/cleanup.go`](../../pkg/convert-linux/hypervisor/plugins/xen/cleanup.go)

| Action | Details |
|--------|---------|
| Modules removed from sysconfig | `xennet`, `xen-vnif`, `xenblk`, `xen-vbd` removed from `INITRD_MODULES` and `DOMU_INITRD_MODULES` in `/etc/sysconfig/kernel` |

### Kudzu

**Code:** [`pkg/convert-linux/hypervisor/plugins/kudzu/cleanup.go`](../../pkg/convert-linux/hypervisor/plugins/kudzu/cleanup.go)

| Action | Details |
|--------|---------|
| Service disabled | `kudzu.service` |
| Files removed | `/etc/rc*.d/[SK]*kudzu` symlinks |

### Cleanup summary

| Hypervisor | Status |
|------------|--------|
| VMware | Yes -- services, repos, dirs, packages |
| Hyper-V | Yes -- services only |
| Citrix/XenServer | Yes -- services, files, inittab |
| AWS EC2 | Yes -- services masked, cloud-init |
| Nutanix AHV | Yes -- services masked, init script, NGT dir |
| Parallels | Yes -- services, dirs |
| VirtualBox | Yes -- services, dirs |
| Xen | Yes -- kernel module refs |
| Kudzu | Yes -- service, rc.d symlinks |

---

## Linux: Install (virtio drivers + guest agent)

Driver and guest agent installation depends on the distro family, not the
source hypervisor.

**Distro handler interface:** [`pkg/convert-linux/distro/distro.go`](../../pkg/convert-linux/distro/distro.go)
**Distro handler plugins:** [`pkg/convert-linux/distro/plugins/`](../../pkg/convert-linux/distro/plugins/)

### Supported distro families

| Family | Distro IDs Matched | Pkg Format | Pkg Manager | Cleanup | Install | Status |
|--------|-------------------|------------|-------------|---------|---------|--------|
| RHEL ([handler](../../pkg/convert-linux/distro/plugins/rhel/rhel.go)) | `rhel`, `centos`, `rocky`, `almalinux`, `ol`, `fedora`, `amzn` | rpm | dnf/yum | Yes | Yes | Yes |
| Debian ([handler](../../pkg/convert-linux/distro/plugins/debian/debian.go)) | `debian`, `ubuntu` | deb | apt | Yes | Yes | Yes |
| SUSE ([handler](../../pkg/convert-linux/distro/plugins/suse/suse.go)) | `sles`, `opensuse-leap`, `opensuse-tumbleweed` | rpm | zypper | Yes | Yes | Yes |
| ALT (no handler) | -- | rpm | apt | Yes | **Stub** | **Partial** |

ALT Linux: Hypervisor cleanup runs normally (it is distro-independent), but
distro-specific install operations (initramfs rebuild, package install, kernel
scanning) fall back to defaults. The pipeline logs a warning when an ALT Linux
guest is detected.

### What gets installed

**VirtIO initramfs injection:** [`pkg/convert-linux/initramfs/virtio.go`](../../pkg/convert-linux/initramfs/virtio.go)
**Guest agent installation:** [`pkg/convert-linux/guestagent/install.go`](../../pkg/convert-linux/guestagent/install.go)

| Component | Details |
|-----------|---------|
| VirtIO kernel modules in initramfs | `virtio`, `virtio_ring`, `virtio_blk`, `virtio_scsi`, `virtio_net`, `virtio_pci`, `xts`, `bochs-drm`, `bochs` |
| Modprobe aliases (`/etc/modprobe.d/kc-virtio.conf`) | `scsi_hostadapter` -> `virtio_blk`, `scsi_hostadapter1` -> `virtio_scsi`, `eth0` -> `virtio_net` |
| Initramfs rebuild | `dracut` (rpm-based) or `update-initramfs`/`mkinitramfs` (deb-based) |
| Guest agent | `qemu-guest-agent` -- local package from `/usr/share/kc-packages/` or firstboot network install (see Amazon Linux below) |

### What gets cleaned up (install-related)

**Modprobe cleanup:** [`pkg/convert-linux/guestcleanup/modalias.go`](../../pkg/convert-linux/guestcleanup/modalias.go)
**Cache cleanup:** [`pkg/convert-linux/guestcleanup/cache.go`](../../pkg/convert-linux/guestcleanup/cache.go)
**SELinux relabel:** [`pkg/convert-linux/selinux/`](../../pkg/convert-linux/selinux/)

| Component | Details |
|-----------|---------|
| Stale modprobe entries removed | `vmw_pvscsi`, `vmxnet3`, `vmxnet`, `hv_vmbus`, `hv_storvsc`, `hv_netvsc`, `xen_blkfront`, `xen_netfront`, `vboxguest`, `vboxsf`, `vboxvideo`, `prl_tg`, `prl_eth`, `prl_fs`, `prl_fs_freeze` |
| SELinux relabeling | Offline `setfiles` (avoids boot-time full relabel) |
| Cache cleanup | `/etc/blkid.tab`, LVM cache, RPM DB locks |

### Guest agent install method by distro

| Pkg Manager | Local install | Network install (firstboot) |
|-------------|--------------|---------------------------|
| dnf | `rpm -ivh` from `/usr/share/kc-packages/rpm/el{N}/{arch}/` | `dnf install -y qemu-guest-agent \|\| yum install -y qemu-guest-agent` |
| apt | `dpkg -i` from `/usr/share/kc-packages/` | `apt-get install -y qemu-guest-agent` |
| zypper | `rpm -ivh` from `/usr/share/kc-packages/` | `zypper --non-interactive install qemu-guest-agent` |

**Amazon Linux (`amzn`):** `VERSION_ID` maps to an EL major for local lookup (`2` → `el7`, `2023` → `el9`) via `localPackageMajorVersion` in [`guestagent/elmajor.go`](../../pkg/convert-linux/guestagent/elmajor.go), then follows the normal local-first/network-fallback flow above — with one exception: Amazon Linux 2023+ never falls back to network install, because `qemu-guest-agent` is not packaged in AL2023 guest repos at all. In practice this means Amazon Linux 2 (no `el7` package staged) always installs via `dnf install` at first boot, while Amazon Linux 2023+ (`el9` package staged in `kc-v2v`) always installs from the bundled local RPM and is skipped entirely if that RPM is missing.

### systemd-networkd guests (blocks 11b / 15)

**Code:** [`pkg/convert-linux/network/networkd/`](../../pkg/convert-linux/network/networkd/)

After hypervisor cleanup, the pipeline calls `networkd.Detect` once. When true:

| Block | Action |
|-------|--------|
| 11b | Writes `10-kc-virtio.network` (virtio DHCP) and a `systemd-networkd-wait-online` drop-in (30s `--any`) |
| 15 | Writes MAC-matched `10-kc-static-*.network` files for plan static IPs (skips `nicnaming` + [`staticip`](../../pkg/convert-linux/network/staticip/) firstboot) |

`Detect` is true when any of: vendor `80-ec2.network`, `ID=amzn` with `VERSION_ID=2023` in os-release (Amazon Linux 2023 only — AL2 falls through), or systemd-networkd enabled without active NetworkManager.

### NIC naming handlers

**Plugin directory:** [`pkg/convert-linux/nicnaming/plugins/`](../../pkg/convert-linux/nicnaming/plugins/)

| Plugin | Used By |
|--------|---------|
| `ifcfg` | RHEL/CentOS (ifcfg-ethN scripts) |
| `nm` | NetworkManager connections |
| `nmdhcp` | NetworkManager DHCP |
| `dhclient` | dhclient config |
| `netplan` | Ubuntu/Debian netplan YAML |
| `wicked` | SUSE Wicked |

Guests matching `networkd.Detect` bypass these plugins for static IP configuration; see **systemd-networkd guests** above. Guests that use `nicnaming` also use [`pkg/convert-linux/network/staticip/`](../../pkg/convert-linux/network/staticip/) in the same pipeline block to write a macToIP mapping file and firstboot `nmcli`/`ip` commands.

---

## Cross-Reference Matrices

### Linux: Cleanup status by hypervisor

Cleanup is distro-independent -- every hypervisor plugin runs the same way
regardless of distro family.

| Hypervisor | RHEL family | Debian family | SUSE family | ALT |
|------------|-------------|---------------|-------------|-----|
| VMware | Full | Full | Full | Full |
| Hyper-V | Full | Full | Full | Full |
| Citrix | Full | Full | Full | Full |
| EC2 | Full | Full | Full | Full |
| Nutanix | Full | Full | Full | Full |
| Parallels | Full | Full | Full | Full |
| VirtualBox | Full | Full | Full | Full |
| Xen | Full | Full | Full | Full |
| Kudzu | Full | Full | Full | Full |

### Linux: Install status by distro

Install is hypervisor-independent -- the same drivers and GA are installed
regardless of the source hypervisor.

| Distro Family | VirtIO Modules | Initramfs Rebuild | Guest Agent | Modprobe Aliases | Overall |
|---------------|---------------|-------------------|-------------|-----------------|---------|
| RHEL | Full | Full (dracut) | Full (dnf/yum) | Full | Yes |
| Debian | Full | Full (update-initramfs) | Full (apt) | Full | Yes |
| SUSE | Full | Full (dracut) | Full (zypper) | Full | Yes |
| ALT | Full | Stub (defaults) | Stub (defaults) | Full | **Stub** |

## Gaps and Notes

1. **ALT Linux install** -- `FamilyALT` constant defined in
   [`pkg/common/types/types.go`](../../pkg/common/types/types.go) but no distro
   handler plugin exists under
   [`pkg/convert-linux/distro/plugins/`](../../pkg/convert-linux/distro/plugins/).
   Package format (`rpm`) and manager (`apt`) are recognized in
   [`pkg/convert-linux/distro/distro.go`](../../pkg/convert-linux/distro/distro.go);
   the pipeline logs a specific warning when an ALT Linux guest is detected.
   Cleanup runs normally.

2. **No unlogged stubs** -- All stubs (ALT Linux distro, Windows Citrix/
   Parallels removal) log warnings at runtime when detected.

---

## Related docs

- [conversion-paths.md](conversion-paths.md) — overview (cleanup vs install)
- [conversion-paths-windows.md](conversion-paths-windows.md) — Windows paths
- [guest-os-handlers.md](guest-os-handlers.md) — distro handler classification
- [../apps/kc-convert-linux.md](../apps/kc-convert-linux.md) — Linux converter pipeline blocks
