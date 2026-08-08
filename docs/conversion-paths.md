# Conversion Paths Reference

This document maps every OS + source-hypervisor code path in kc-utils.
Conversion has two independent concerns:

- **Cleanup** -- remove/disable drivers, services, packages, and registry
  entries left by the source hypervisor. Depends on which hypervisor the VM
  comes from.
- **Install** -- install virtio drivers and guest agent so the VM boots on
  KubeVirt/KVM. Depends on the target OS.

These two concerns are tracked separately below.

**Pipeline entry points:**
- Linux: [`pkg/cmd/convert-linux/pipeline.go`](../pkg/cmd/convert-linux/pipeline.go)
- Windows: [`pkg/cmd/convert-windows/pipeline.go`](../pkg/cmd/convert-windows/pipeline.go)

---

## Linux: Cleanup (source hypervisor removal)

Every Linux hypervisor cleanup plugin runs independently of the distro.
Each plugin detects its own artifacts and cleans them up.

**Plugin interface:** [`pkg/convert-linux/hypervisor/hypervisor.go`](../pkg/convert-linux/hypervisor/hypervisor.go)
**Plugin directory:** [`pkg/convert-linux/hypervisor/plugins/`](../pkg/convert-linux/hypervisor/plugins/)

### VMware

**Code:** [`pkg/convert-linux/hypervisor/plugins/vmware/cleanup.go`](../pkg/convert-linux/hypervisor/plugins/vmware/cleanup.go)

| Action | Details |
|--------|---------|
| Services disabled | `vmtoolsd.service`, `open-vm-tools.service`, `vgauthd.service` |
| Repos disabled | All yum `.repo` files containing `vmware.com` set to `enabled=0` |
| Directories removed | `/etc/vmware-tools`, `/usr/lib/vmware-tools`, `/usr/lib64/vmware-tools` |
| Packages removed (firstboot) | `open-vm-tools`, `open-vm-tools-desktop`, `VMwareTools` via oneshot systemd unit |

### Hyper-V

**Code:** [`pkg/convert-linux/hypervisor/plugins/hyperv/cleanup.go`](../pkg/convert-linux/hypervisor/plugins/hyperv/cleanup.go)

| Action | Details |
|--------|---------|
| Services disabled | `hv-kvp-daemon.service`, `hv-fcopy-daemon.service`, `hv-vss-daemon.service`, `hypervkvpd.service`, `hypervfcopyd.service`, `hypervvssd.service` |
| Files removed | None |
| Packages removed | None |

### Citrix/XenServer

**Code:** [`pkg/convert-linux/hypervisor/plugins/citrix/cleanup.go`](../pkg/convert-linux/hypervisor/plugins/citrix/cleanup.go)

| Action | Details |
|--------|---------|
| Services disabled | `xe-daemon.service`, `xapi.service` |
| Files removed | `/etc/xensource-inventory`, `/usr/sbin/xe-daemon` |
| Extra | Restores commented-out getty lines in `/etc/inittab` |

### AWS EC2

**Code:** [`pkg/convert-linux/hypervisor/plugins/ec2/cleanup.go`](../pkg/convert-linux/hypervisor/plugins/ec2/cleanup.go)

| Action | Details |
|--------|---------|
| Service symlinks removed | `amazon-ssm-agent.service`, `amazon-cloudwatch-agent.service`, `ec2-instance-connect.service` |
| Cloud-init | Disabled via `99-kc-disable-ec2.cfg` (`datasource_list: [None]`) |

### Nutanix AHV

**Code:** [`pkg/convert-linux/hypervisor/plugins/nutanix/cleanup.go`](../pkg/convert-linux/hypervisor/plugins/nutanix/cleanup.go)

| Action | Details |
|--------|---------|
| Service symlinks removed | `ngt_guest_agent.service` |
| Files removed | `/etc/rc.d/init.d/ngt_guest_agent` |

### Parallels

**Code:** [`pkg/convert-linux/hypervisor/plugins/parallels/cleanup.go`](../pkg/convert-linux/hypervisor/plugins/parallels/cleanup.go)

| Action | Details |
|--------|---------|
| Services disabled | `prltoolsd.service`, `prl-xorg-cleanup.service` |
| Directories removed | `/usr/lib/parallels-tools`, `/usr/lib64/parallels-tools` |

### VirtualBox

**Code:** [`pkg/convert-linux/hypervisor/plugins/virtualbox/cleanup.go`](../pkg/convert-linux/hypervisor/plugins/virtualbox/cleanup.go)

| Action | Details |
|--------|---------|
| Services disabled | `vboxadd-service.service`, `vboxadd.service`, `vboxservice.service` |
| Directories removed | Custom install dir (from config), `/var/lib/VBoxGuestAdditions`, `/opt/VBoxGuestAdditions` |

### Xen (kernel modules)

**Code:** [`pkg/convert-linux/hypervisor/plugins/xen/cleanup.go`](../pkg/convert-linux/hypervisor/plugins/xen/cleanup.go)

| Action | Details |
|--------|---------|
| Modules removed from sysconfig | `xennet`, `xen-vnif`, `xenblk`, `xen-vbd` removed from `INITRD_MODULES` and `DOMU_INITRD_MODULES` in `/etc/sysconfig/kernel` |

### Kudzu

**Code:** [`pkg/convert-linux/hypervisor/plugins/kudzu/cleanup.go`](../pkg/convert-linux/hypervisor/plugins/kudzu/cleanup.go)

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
| AWS EC2 | Yes -- service symlinks, cloud-init |
| Nutanix AHV | Yes -- service symlinks, files |
| Parallels | Yes -- services, dirs |
| VirtualBox | Yes -- services, dirs |
| Xen | Yes -- kernel module refs |
| Kudzu | Yes -- service, rc.d symlinks |

---

## Linux: Install (virtio drivers + guest agent)

Driver and guest agent installation depends on the distro family, not the
source hypervisor.

**Distro handler interface:** [`pkg/convert-linux/distro/distro.go`](../pkg/convert-linux/distro/distro.go)
**Distro handler plugins:** [`pkg/convert-linux/distro/plugins/`](../pkg/convert-linux/distro/plugins/)

### Supported distro families

| Family | Distro IDs Matched | Pkg Format | Pkg Manager | Cleanup | Install | Status |
|--------|-------------------|------------|-------------|---------|---------|--------|
| RHEL ([handler](../pkg/convert-linux/distro/plugins/rhel/rhel.go)) | `rhel`, `centos`, `rocky`, `almalinux`, `ol`, `fedora`, `amzn` | rpm | dnf/yum | Yes | Yes | Yes |
| Debian ([handler](../pkg/convert-linux/distro/plugins/debian/debian.go)) | `debian`, `ubuntu` | deb | apt | Yes | Yes | Yes |
| SUSE ([handler](../pkg/convert-linux/distro/plugins/suse/suse.go)) | `sles`, `opensuse-leap`, `opensuse-tumbleweed` | rpm | zypper | Yes | Yes | Yes |
| ALT (no handler) | -- | rpm | apt | Yes | **Stub** | **Partial** |

ALT Linux: Hypervisor cleanup runs normally (it is distro-independent), but
distro-specific install operations (initramfs rebuild, package install, kernel
scanning) fall back to defaults. The pipeline logs a warning when an ALT Linux
guest is detected.

### What gets installed

**VirtIO initramfs injection:** [`pkg/convert-linux/initramfs/virtio.go`](../pkg/convert-linux/initramfs/virtio.go)
**Guest agent installation:** [`pkg/convert-linux/guestagent/install.go`](../pkg/convert-linux/guestagent/install.go)

| Component | Details |
|-----------|---------|
| VirtIO kernel modules in initramfs | `virtio`, `virtio_ring`, `virtio_blk`, `virtio_scsi`, `virtio_net`, `virtio_pci`, `xts`, `bochs-drm`, `bochs` |
| Modprobe aliases (`/etc/modprobe.d/kc-virtio.conf`) | `scsi_hostadapter` -> `virtio_blk`, `scsi_hostadapter1` -> `virtio_scsi`, `eth0` -> `virtio_net` |
| Initramfs rebuild | `dracut` (rpm-based) or `update-initramfs`/`mkinitramfs` (deb-based) |
| Guest agent | `qemu-guest-agent` -- local package from `/usr/share/kc-packages/` or firstboot network install |

### What gets cleaned up (install-related)

**Modprobe cleanup:** [`pkg/convert-linux/guestcleanup/modalias.go`](../pkg/convert-linux/guestcleanup/modalias.go)
**Cache cleanup:** [`pkg/convert-linux/guestcleanup/cache.go`](../pkg/convert-linux/guestcleanup/cache.go)
**SELinux relabel:** [`pkg/convert-linux/selinux/`](../pkg/convert-linux/selinux/)

| Component | Details |
|-----------|---------|
| Stale modprobe entries removed | `vmw_pvscsi`, `vmxnet3`, `vmxnet`, `hv_vmbus`, `hv_storvsc`, `hv_netvsc`, `xen_blkfront`, `xen_netfront`, `vboxguest`, `vboxsf`, `vboxvideo` |
| SELinux relabeling | Offline `setfiles` (avoids boot-time full relabel) |
| Cache cleanup | `/etc/blkid.tab`, LVM cache, RPM DB locks |

### Guest agent install method by distro

| Pkg Manager | Local install | Network install (firstboot) |
|-------------|--------------|---------------------------|
| dnf | `rpm -ivh` from `/usr/share/kc-packages/rpm/el{N}/{arch}/` | `dnf install -y qemu-guest-agent \|\| yum install -y qemu-guest-agent` |
| apt | `dpkg -i` from `/usr/share/kc-packages/` | `apt-get install -y qemu-guest-agent` |
| zypper | `rpm -ivh` from `/usr/share/kc-packages/` | `zypper --non-interactive install qemu-guest-agent` |

### NIC naming handlers

**Plugin directory:** [`pkg/convert-linux/nicnaming/plugins/`](../pkg/convert-linux/nicnaming/plugins/)

| Plugin | Used By |
|--------|---------|
| `ifcfg` | RHEL/CentOS (ifcfg-ethN scripts) |
| `nm` | NetworkManager connections |
| `nmdhcp` | NetworkManager DHCP |
| `dhclient` | dhclient config |
| `netplan` | Ubuntu/Debian netplan YAML |
| `wicked` | SUSE Wicked |

---

## Windows: Cleanup (source hypervisor removal)

Windows hypervisor cleanup has two pipeline phases: software removal (block 4)
and service disabling (block 8). Each phase has its own set of plugins.

**Plugin interface:** [`pkg/convert-windows/hypervisor/hypervisor.go`](../pkg/convert-windows/hypervisor/hypervisor.go)
**Remove plugins:** [`pkg/convert-windows/hypervisor/plugins/remove/`](../pkg/convert-windows/hypervisor/plugins/remove/)
**Service plugins:** [`pkg/convert-windows/hypervisor/plugins/services/`](../pkg/convert-windows/hypervisor/plugins/services/)

### VMware

**Remove:** [`plugins/remove/vmware/remove.go`](../pkg/convert-windows/hypervisor/plugins/remove/vmware/remove.go)
**Services:** [`plugins/services/vmware/services.go`](../pkg/convert-windows/hypervisor/plugins/services/vmware/services.go)
**Firstboot:** [`firstboot/plugins/vmwarecleanup/`](../pkg/convert-windows/firstboot/plugins/vmwarecleanup/)

**Software removal:**

| Action | Details |
|--------|---------|
| Directories removed | `Program Files\VMware\VMware Tools` |
| Registry keys removed | `Uninstall\VMware Tools`, MSI Installer product/feature entries, VMware scheduled tasks |
| Firstboot script | `msiexec /x {GUID}` per product, WMI uninstall of remaining products, stop+delete residual services |

**Services disabled (Start=4):**
`VMTools`, `VGAuthService`, `VMwarePhysicalDiskHelper`, `vm3dservice`, `VMUSBArbService`

**Firstboot driver cleanup (priority 9100):**
PNP removal (Win8+) or devcon-based removal (Win7 and older) of residual
VMware drivers.

### AWS PV Drivers

**Remove:** [`plugins/remove/awspv/remove.go`](../pkg/convert-windows/hypervisor/plugins/remove/awspv/remove.go)

| Action | Details |
|--------|---------|
| Registry keys removed | `Uninstall\AWS PV Drivers` |
| Driver files removed | All `xen*.sys` from `System32\drivers` |

### EC2/Amazon

**Remove:** [`plugins/remove/ec2/cleanup.go`](../pkg/convert-windows/hypervisor/plugins/remove/ec2/cleanup.go)

| Action | Details |
|--------|---------|
| Services disabled (Start=4) | `AWSPVDrivers`, `Xennet`, `XenVbd`, `XenVif`, `AWSNVME`, `AmazonSSMAgent`, `AmazonCloudWatchAgent`, `Ec2Config`, `EC2Launch` |
| Scheduled tasks removed | Amazon EC2 Launch tasks |
| Driver files removed | `xen*.sys` |

### EC2 Launch

**Remove:** [`plugins/remove/ec2launch/remove.go`](../pkg/convert-windows/hypervisor/plugins/remove/ec2launch/remove.go)

| Action | Details |
|--------|---------|
| Registry keys removed | EC2Launch/Ec2Config uninstall keys |
| Directories removed | `Program Files\Amazon\EC2Launch`, `Program Files\Amazon\Ec2ConfigService` |

### Nutanix

**Remove:** [`plugins/remove/nutanix/remove.go`](../pkg/convert-windows/hypervisor/plugins/remove/nutanix/remove.go)
**Services:** [`plugins/services/nutanix/services.go`](../pkg/convert-windows/hypervisor/plugins/services/nutanix/services.go)

**Software removal:**

| Action | Details |
|--------|---------|
| Directories removed | `Program Files\Nutanix`, `Program Files (x86)\Nutanix` |
| Registry keys removed | `Uninstall\Nutanix Guest Tools` |

**Services disabled (Start=4):**
`NutanixGuestTools`, `NutanixGuestAgent`, `NgtService`

### VirtualBox

**Remove:** [`plugins/remove/virtualbox/remove.go`](../pkg/convert-windows/hypervisor/plugins/remove/virtualbox/remove.go)
**Services:** [`plugins/services/virtualbox/services.go`](../pkg/convert-windows/hypervisor/plugins/services/virtualbox/services.go)

**Software removal:**

| Action | Details |
|--------|---------|
| Directories removed | `Program Files\Oracle\VirtualBox Guest Additions` |
| Registry keys removed | `Uninstall\Oracle VM VirtualBox Guest Additions` |
| Driver files removed | All `vbox*.sys` from `System32\drivers` |

**Services disabled (Start=4):**
`VBoxService`, `VBoxGuest`, `VBoxSF`, `VBoxVideo`, `VBoxMouse`

### Hyper-V

**Remove:** [`plugins/remove/hyperv/remove.go`](../pkg/convert-windows/hypervisor/plugins/remove/hyperv/remove.go)
**Services:** [`plugins/services/hyperv/services.go`](../pkg/convert-windows/hypervisor/plugins/services/hyperv/services.go)

Hyper-V integration components are inbox Windows drivers and cannot be safely
removed offline. The remove plugin logs their presence; the services plugin
disables them.

**Services disabled (Start=4):**
`vmicheartbeat`, `vmicshutdown`, `vmicexchange`, `vmicvss`, `vmictimesync`,
`vmicrdv`, `vmicguestinterface`, `vmickvpexchange`

**Inbox drivers left in place:**
`vmbus.sys`, `storvsc.sys`, `netvsc.sys`, `VMBusHID.sys`, `hypervideo.sys`

### Citrix/XenServer -- Stub

**Remove:** [`plugins/remove/citrix/remove.go`](../pkg/convert-windows/hypervisor/plugins/remove/citrix/remove.go)

Detects Citrix XenTools via `Program Files\Citrix\XenTools` directory or
`Uninstall\Citrix XenTools` registry key.
Logs a warning that removal is not yet implemented; no cleanup is performed.

### Parallels -- Stub

**Remove:** [`plugins/remove/parallels/remove.go`](../pkg/convert-windows/hypervisor/plugins/remove/parallels/remove.go)

Detects Parallels Tools via `Program Files\Parallels\Parallels Tools` directory
or `Uninstall\Parallels Tools` registry key.
Logs a warning that removal is not yet implemented; no cleanup is performed.

### Cleanup summary

| Hypervisor | Software Removal | Service Disable | Status |
|------------|-----------------|-----------------|--------|
| VMware | Yes | Yes | Yes |
| AWS PV | Yes | -- | Yes |
| EC2/Amazon | Yes | -- | Yes |
| EC2 Launch | Yes | -- | Yes |
| Nutanix | Yes | Yes | Yes |
| VirtualBox | Yes | Yes | Yes |
| Hyper-V | Yes | Yes | Yes |
| Citrix | **Stub** (detect + warn) | None | **Stub** |
| Parallels | **Stub** (detect + warn) | None | **Stub** |

---

## Windows: Install (virtio drivers + guest agent)

Driver and guest agent installation depends on the Windows version, not the
source hypervisor.

**Version handler interface:** [`pkg/convert-windows/version/version.go`](../pkg/convert-windows/version/version.go)
**Version handlers:** [`pkg/convert-windows/version/handlers.go`](../pkg/convert-windows/version/handlers.go)
**Version registration:** [`pkg/convert-windows/version/register.go`](../pkg/convert-windows/version/register.go)

### Supported versions

| Handler | Matches | Cleanup | Install | Status |
|---------|---------|---------|---------|--------|
| win11 | Win 11, Server 2022, Server 2025 | Yes | Yes | Yes |
| win10 | Win 10, Server 2016, Server 2019 | Yes | Yes | Yes |
| win81 | Win 8.1, Server 2012 R2 | Yes | Yes | Yes |
| win8 | Win 8, Server 2012 | Yes | Yes | Yes |
| win7 | Windows 7 | Yes | Yes | Yes |
| win2008r2 | Server 2008 R2 | Yes | Yes | Yes |
| win2008 | Server 2008 | Yes | **Partial** (no GA) | **Partial** |
| winvista | Vista | Yes | **Partial** (no GA) | **Partial** |
| win2003 | Server 2003 | Yes | **Partial** (no GA, no PS) | **Partial** |
| winxp | Windows XP | Yes | **Partial** (no GA, no PS) | **Partial** |

### Version capabilities

| Handler | VirtIO Dir | Registrar | Launcher | PowerShell | GA MSI | NTFS Fix |
|---------|-----------|-----------|----------|------------|--------|----------|
| win11 | `w11` | driverdb | Modern | Yes | Yes | No |
| win10 | `w10` | driverdb | Modern | Yes | Yes | No |
| win81 | `w8.1` | driverdb | Modern | Yes | Yes | No |
| win8 | `w8` | driverdb | Modern | Yes | Yes | No |
| win7 | `w7` | criticaldb | PSV1 | Yes | Yes | No |
| win2008r2 | `2k8r2` | criticaldb | PSV1 | Yes | Yes | No |
| win2008 | `2k8` | criticaldb | PSV1 | Yes | **No** | No |
| winvista | `vista` | criticaldb | PSV1 | Yes | **No** | No |
| win2003 | `2k3` | criticaldb | BatOnly | **No** | **No** | Yes |
| winxp | `xp` | criticaldb | BatOnly | **No** | **No** | Yes |

**Driver registrars:**
- driverdb (Win8+): [`pkg/convert-windows/drivers/plugins/driverdb/`](../pkg/convert-windows/drivers/plugins/driverdb/)
- criticaldb (Win7 and older): [`pkg/convert-windows/drivers/plugins/criticaldb/`](../pkg/convert-windows/drivers/plugins/criticaldb/)

**Guest agent exclusions:** [`pkg/convert-windows/version/guestagent.go`](../pkg/convert-windows/version/guestagent.go)

### What gets installed

**Driver copy:** [`pkg/convert-windows/drivers/copy.go`](../pkg/convert-windows/drivers/copy.go)
**Driver source/collection:** [`pkg/convert-windows/driversource/`](../pkg/convert-windows/driversource/)
**DevicePath update:** [`pkg/convert-windows/drivers/devicepath.go`](../pkg/convert-windows/drivers/devicepath.go)
**Crash control:** [`pkg/convert-windows/crashcontrol/`](../pkg/convert-windows/crashcontrol/)

| Component | Details |
|-----------|---------|
| VirtIO drivers copied | `.sys`, `.inf`, `.cat` files to `C:\Windows\Drivers\VirtIO\` |
| Boot-critical drivers | `viostor.sys`, `vioscsi.sys` also copied to `System32\drivers\` |
| Driver registration | Service entries with `Start=0` (boot start) for `viostor`/`vioscsi` |
| DevicePath | `%SystemRoot%\Drivers\VirtIO` appended to registry `DevicePath` |
| Crash control | Disables auto-reboot on BSOD (`AutoReboot=0`) |
| NTFS heads fix | Win2003 and WinXP only |

### Firstboot install scripts

**Firstboot framework:** [`pkg/convert-windows/firstboot/`](../pkg/convert-windows/firstboot/)
**Firstboot plugins:** [`pkg/convert-windows/firstboot/plugins/`](../pkg/convert-windows/firstboot/plugins/)

| Priority | Component | Description |
|----------|-----------|-------------|
| 2000 | [pnputil](../pkg/convert-windows/firstboot/plugins/pnputil/) | Install VirtIO drivers via PnP |
| 2500 | [staticip](../pkg/convert-windows/firstboot/plugins/staticipfb/) | Reconfigure static IPs (method varies by OS version) |
| 2600 | [routecleanup](../pkg/convert-windows/firstboot/plugins/routecleanup/) | Remove duplicate persistent routes |
| 2700 | [multipleips](../pkg/convert-windows/firstboot/plugins/multipleips/) | Add secondary IPs per NIC |
| 3000 | [qemuga](../pkg/convert-windows/firstboot/plugins/qemuga/) | Install QEMU guest agent MSI (Win7+ only) |
| 4000 | [diskonliner](../pkg/convert-windows/firstboot/plugins/diskonliner/) | Bring offline disks online |
| 9100 | [vmwarecleanup](../pkg/convert-windows/firstboot/plugins/vmwarecleanup/) | Remove residual VMware drivers/services |
| 99999 | [signal](../pkg/convert-windows/firstboot/plugins/signal/) | Signal conversion completion on COM1 |

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

### Windows: Cleanup status by hypervisor

Cleanup is version-independent -- every hypervisor plugin runs the same way
regardless of Windows version.

| Hypervisor | All Windows versions |
|------------|---------------------|
| VMware | Full |
| AWS PV | Full |
| EC2/Amazon | Full |
| EC2 Launch | Full |
| Nutanix | Full |
| VirtualBox | Full |
| Hyper-V | Full |
| Citrix | Stub (detect + warn) |
| Parallels | Stub (detect + warn) |

### Windows: Install status by version

Install is hypervisor-independent -- the same VirtIO drivers are installed
regardless of the source hypervisor.

| Version | VirtIO Drivers | Driver Registration | Guest Agent MSI | Overall |
|---------|---------------|--------------------|-----------------|---------| 
| Win11 / Srv2022 / Srv2025 | Full | Full (driverdb) | Full | Yes |
| Win10 / Srv2016 / Srv2019 | Full | Full (driverdb) | Full | Yes |
| Win8.1 / Srv2012R2 | Full | Full (driverdb) | Full | Yes |
| Win8 / Srv2012 | Full | Full (driverdb) | Full | Yes |
| Win7 | Full | Full (criticaldb) | Full | Yes |
| Srv2008R2 | Full | Full (criticaldb) | Full | Yes |
| Srv2008 | Full | Full (criticaldb) | **No** | **Partial** |
| Vista | Full | Full (criticaldb) | **No** | **Partial** |
| Srv2003 | Full | Full (criticaldb) | **No** | **Partial** |
| WinXP | Full | Full (criticaldb) | **No** | **Partial** |

---

## Gaps and Notes

1. **ALT Linux install** -- `FamilyALT` constant defined in
   [`pkg/common/types/types.go`](../pkg/common/types/types.go) but no distro
   handler plugin exists under
   [`pkg/convert-linux/distro/plugins/`](../pkg/convert-linux/distro/plugins/).
   Package format (`rpm`) and manager (`apt`) are recognized in
   [`pkg/convert-linux/distro/distro.go`](../pkg/convert-linux/distro/distro.go);
   the pipeline logs a specific warning when an ALT Linux guest is detected.
   Cleanup runs normally.

2. **Windows Citrix/Parallels cleanup** -- Stub removal plugins exist
   under [`pkg/convert-windows/hypervisor/plugins/remove/`](../pkg/convert-windows/hypervisor/plugins/remove/)
   that detect the hypervisor's presence and log a warning, but no actual
   cleanup is performed. The Linux side fully handles both. Install is
   unaffected (VirtIO drivers install normally regardless of source hypervisor).

3. **Guest agent exclusions** -- `win2008`, `winvista`, `win2003`, and `winxp`
   do not receive the QEMU guest agent MSI (see
   [`pkg/convert-windows/version/guestagent.go`](../pkg/convert-windows/version/guestagent.go)).
   VirtIO driver installation works normally for these versions.

4. **No unlogged stubs** -- All stubs (ALT Linux distro, Windows Citrix/
   Parallels removal) log warnings at runtime when detected.
