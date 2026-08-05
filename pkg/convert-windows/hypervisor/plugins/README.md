# hypervisor plugins

Two plugin registries in this block:

## WindowsRemove

Remove hypervisor-specific software from the guest filesystem.
`Detect` uses the SOFTWARE hive for Uninstall keys (SYSTEM alone is not enough).

| Key | Package | Hypervisor |
|-----|---------|------------|
| `vmware` | remove/vmware/ | VMware Tools |
| `nutanix` | remove/nutanix/ | Nutanix guest tools |
| `awspv` | remove/awspv/ | AWS PV drivers |
| `ec2launch` | remove/ec2launch/ | EC2Launch agent |
| `ec2` | remove/ec2/ | EC2 cloud-init cleanup |
| `virtualbox` | remove/virtualbox/ | VirtualBox guest additions |

## WindowsServiceDisabler

Disable hypervisor services via registry (`Start=4`).

| Key | Package | Hypervisor |
|-----|---------|------------|
| `vmware` | services/vmware/ | VMware services |
| `nutanix` | services/nutanix/ | Nutanix services |
| `virtualbox` | services/virtualbox/ | VirtualBox services |
