# hypervisor plugins

`LinuxCleanup` interface — remove or disable source hypervisor agents and tools.

After conversion to KVM the source hypervisor's guest tools are unnecessary and
can interfere with normal operation (e.g. VMware Tools trying to communicate
with a non-existent host daemon, or Xen PV modules failing to load). Each
cleanup plugin detects whether its hypervisor's artifacts are present on the
guest and removes them.

Cleanup goes beyond systemd wants-symlinks: packages/tarball uninstallers when present,
unit disable, and known path removal. Xen scrub and kudzu disable are included.
Every plugin implements `Detect(mountRoot) bool` to check for the presence of
hypervisor artifacts, and `Cleanup(mountRoot) error` to remove them. Errors are
non-fatal — they are recorded as block warnings but do not abort the pipeline.

| Key | Package | Hypervisor |
|-----|---------|------------|
| `vmware` | vmware/ | VMware Tools (`vmware-uninstall-tools.pl`, packages, paths) |
| `virtualbox` | virtualbox/ | VirtualBox Guest Additions (`uninstall.sh`, units, paths) |
| `citrix` | citrix/ | Citrix XenServer / xe tools |
| `parallels` | parallels/ | Parallels Tools |
| `xen` | xen/ | Xen modules / `sysconfig/kernel` scrub |
| `kudzu` | kudzu/ | Disable kudzu (old RHEL) |
| `hyperv` | hyperv/ | Hyper-V daemons |
| `ec2` | ec2/ | Amazon EC2 cloud-init / agent cleanup |
| `nutanix` | nutanix/ | Nutanix guest tools |

## vmware

**What it does:** Removes VMware Tools from the guest filesystem.

**How it works:** Looks for `/usr/bin/vmware-uninstall-tools.pl` (tarball
install) or VMware Tools packages. Disables and removes systemd units
(`vmtoolsd`, `vmware-tools`), deletes known VMware paths
(`/etc/vmware-tools/`, `/usr/lib/vmware-tools/`), and removes VMware-specific
kernel modules from modprobe configuration.

## virtualbox

**What it does:** Removes VirtualBox Guest Additions.

**How it works:** Checks for the VirtualBox uninstaller script or Guest
Additions paths. Disables systemd units (`vboxadd`, `vboxadd-service`),
removes installation directories, and cleans up kernel modules.

## citrix

**What it does:** Removes Citrix XenServer / xe guest tools.

**How it works:** Detects Citrix tools by checking for `xe-*` binaries and
XenServer packages. Removes the tools packages, disables related services,
and cleans up configuration under `/etc/xensource/`.

## parallels

**What it does:** Removes Parallels Tools from the guest.

**How it works:** Detects Parallels installation via known paths and service
units. Disables Parallels services and removes tool binaries and libraries.

## xen

**What it does:** Scrubs Xen-specific kernel module configuration.

**How it works:** Removes Xen module entries from `/etc/sysconfig/kernel`
(SUSE) and modprobe configuration. Ensures Xen PV modules are not loaded
on the converted KVM guest.

## kudzu

**What it does:** Disables the kudzu hardware detection service found on
old RHEL 4/5 systems.

**How it works:** Checks for the `kudzu` service unit or init script and
disables it. Kudzu would otherwise attempt to reconfigure hardware on boot
and potentially interfere with the converted guest's device setup.

## hyperv

**What it does:** Disables Hyper-V Integration Services daemons.

**How it works:** Detects `hv_*` systemd units (e.g. `hv_kvp_daemon`,
`hv_vss_daemon`, `hv_fcopy_daemon`) and disables them. These daemons
communicate with the Hyper-V host and are non-functional under KVM.

## ec2

**What it does:** Cleans up Amazon EC2 cloud-init configuration, agent
artifacts, and migrates Amazon Linux systemd-networkd off EC2 IMDS networking.

**How it works:** Detects EC2-specific cloud-init datasource configuration
and agent packages. Masks EC2 agent systemd units, patches `cloud.cfg` and
`cloud.cfg.d` snippets to `datasource_list: [None]` (not only a drop-in file),
masks EC2-only net hooks via [`systemd.DisableEC2NetHooks`](../../systemd/) (`set-hostname-imds`, policy-route template units),
and leaves agent binaries in place for offline conversion.

For Amazon Linux 2023 and other **systemd-networkd-primary** guests (`Detect`: vendor `80-ec2.network`, `ID=amzn` with `VERSION_ID=2023` in os-release, or networkd enabled without active NetworkManager), the convert-linux pipeline installs virtio DHCP and a wait-online drop-in via [`networkd`](../../network/networkd/). Amazon Linux 2 is not matched unconditionally — it falls through to the networkd-enabled check like any other distro. Static IPs use MAC-matched `.network` files from pipeline block 15 instead of nmcli firstboot.

## nutanix

**What it does:** Removes Nutanix AHV guest tools.

**How it works:** Detects Nutanix guest tools packages and services. Masks
Nutanix-specific systemd units, removes the SysV init script, and deletes
`/usr/local/nutanix/ngt`.
