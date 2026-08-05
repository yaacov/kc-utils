# hypervisor plugins

`LinuxCleanup` interface — remove or disable source hypervisor agents and tools.

Cleanup goes beyond systemd wants-symlinks: packages/tarball uninstallers when present,
unit disable, and known path removal. Xen scrub and kudzu disable are included.

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
