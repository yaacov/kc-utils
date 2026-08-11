# hypervisor plugins

Two plugin registries in this block for removing source hypervisor artifacts
from Windows guests. Removal happens in two phases: first the `WindowsRemove`
plugins delete hypervisor software, tools directories, and registry uninstall
keys from the offline guest filesystem; then the `WindowsServiceDisabler`
plugins set hypervisor service registry entries to `Start=4` (disabled) so
they do not start on the next boot. Both phases operate on the Windows registry
hives (SYSTEM and SOFTWARE) opened via the `pkg/common/registry` package.

## WindowsRemove

Remove hypervisor-specific software from the guest filesystem.
`Detect` uses the SOFTWARE hive for Uninstall keys (SYSTEM alone is not enough).
Each plugin's `Remove` method deletes tool directories from the guest
filesystem, removes uninstall registry keys, and optionally schedules a
firstboot script for cleanup that requires Windows to be running (e.g. MSI
uninstallation via `msiexec`).

| Key | Package | Hypervisor |
|-----|---------|------------|
| `vmware` | remove/vmware/ | VMware Tools |
| `nutanix` | remove/nutanix/ | Nutanix guest tools |
| `awspv` | remove/awspv/ | AWS PV drivers |
| `ec2launch` | remove/ec2launch/ | EC2Launch agent |
| `ec2` | remove/ec2/ | EC2 cloud-init cleanup |
| `virtualbox` | remove/virtualbox/ | VirtualBox guest additions |
| `citrix` | remove/citrix/ | Citrix XenTools / XenServer tools |
| `hyperv` | remove/hyperv/ | Hyper-V Integration Services |
| `parallels` | remove/parallels/ | Parallels Tools |

### vmware

**What it does:** Removes VMware Tools from the Windows guest — the most
complex removal due to VMware's deep integration.

**How it works:** Detects via the `Program Files\VMware\VMware Tools` directory
or the VMware Tools uninstall registry key. Removes the tools directory, cleans
up MSI product entries from `Classes\Installer\Products` and
`UserData\S-1-5-18\Products` in the SOFTWARE hive (decoding Windows Installer
GUIDs), removes scheduled tasks under
`Microsoft\Windows NT\CurrentVersion\Schedule\TaskCache\Tree\VMware`, and
writes a firstboot PowerShell script that runs `msiexec /x {GUID} /qn` for
each found MSI product, queries `Win32_Product` for remaining VMware entries,
and stops/deletes residual VMware services.

### nutanix

**What they do:** Each removes its respective hypervisor's guest tools,
following the same `Detect` → `Remove` pattern — checking for known
directories and registry keys, deleting files, and cleaning up uninstall
entries.

### awspv

**What it does:** Removes AWS PV driver uninstall key, `xen*.sys` files, and
boot-critical `XENFILT` `UpperFilters` entries on System and HDC class GUIDs.
Also disables the `xenfilt` service (`Start=4`).

**How it works:** Uses the SYSTEM hive to clear `UpperFilters` via
[`RemoveFilter`](../../hypervisor/filters.go) and disables `xenfilt` before
deleting driver files — leaving `XENFILT` registered causes
`INACCESSIBLE_BOOT_DEVICE` on next boot.

### ec2

**What it does:** Disables EC2 Xen services, removes `xen*.sys`, clears
`XENFILT` `UpperFilters` (same trap as AWS PV), and disables `xenfilt`.

### ec2launch / virtualbox / citrix / hyperv / parallels

**What they do:** Each removes its respective hypervisor's guest tools,
following the same `Detect` → `Remove` pattern — checking for known
directories and registry keys, deleting files, and cleaning up uninstall
entries.

## WindowsServiceDisabler

Disable hypervisor services via registry (`Start=4`).
Each plugin declares the service names it manages via `ServiceNames()`. The
`DisableServices` method sets `Start=4` (disabled) in the SYSTEM hive for each
service that has an existing registry key under
`{CurrentControlSet}\Services\`.

| Key | Package | Hypervisor |
|-----|---------|------------|
| `vmware` | services/vmware/ | VMware services (`VMTools`, `VGAuthService`, `VMwarePhysicalDiskHelper`, `vm3dservice`, `VMUSBArbService`) |
| `nutanix` | services/nutanix/ | Nutanix services |
| `virtualbox` | services/virtualbox/ | VirtualBox services |
| `hyperv` | services/hyperv/ | Hyper-V Integration Services |
