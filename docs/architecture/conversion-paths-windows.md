# Windows Conversion Paths

Maps every Windows source-hypervisor cleanup path and version install path in kc-utils.

## Windows: Cleanup (source hypervisor removal)

Windows hypervisor cleanup has two pipeline phases: software removal (block 4)
and service disabling (block 8). Each phase has its own set of plugins.

**Plugin interface:** [`pkg/convert-windows/hypervisor/hypervisor.go`](../../pkg/convert-windows/hypervisor/hypervisor.go)
**Remove plugins:** [`pkg/convert-windows/hypervisor/plugins/remove/`](../../pkg/convert-windows/hypervisor/plugins/remove/)
**Service plugins:** [`pkg/convert-windows/hypervisor/plugins/services/`](../../pkg/convert-windows/hypervisor/plugins/services/)

### VMware

**Remove:** [`../../pkg/convert-windows/hypervisor/plugins/remove/vmware/remove.go`](../../pkg/convert-windows/hypervisor/plugins/remove/vmware/remove.go)
**Services:** [`../../pkg/convert-windows/hypervisor/plugins/services/vmware/services.go`](../../pkg/convert-windows/hypervisor/plugins/services/vmware/services.go)
**Firstboot:** [`../../pkg/convert-windows/firstboot/plugins/vmwarecleanup/`](../../pkg/convert-windows/firstboot/plugins/vmwarecleanup/)

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

**Remove:** [`../../pkg/convert-windows/hypervisor/plugins/remove/awspv/remove.go`](../../pkg/convert-windows/hypervisor/plugins/remove/awspv/remove.go)

| Action | Details |
|--------|---------|
| Registry keys removed | `Uninstall\AWS PV Drivers` |
| Driver files removed | All `xen*.sys` from `System32\drivers` |
| UpperFilters cleaned | Remove `XENFILT` from System and HDC class GUIDs |
| Services disabled (Start=4) | `xenfilt` |

### EC2/Amazon

**Remove:** [`../../pkg/convert-windows/hypervisor/plugins/remove/ec2/cleanup.go`](../../pkg/convert-windows/hypervisor/plugins/remove/ec2/cleanup.go)

| Action | Details |
|--------|---------|
| Services disabled (Start=4) | `AWSPVDrivers`, `Xennet`, `XenVbd`, `XenVif`, `AWSNVME`, `AmazonSSMAgent`, `AmazonCloudWatchAgent`, `Ec2Config`, `EC2Launch`, `xenfilt` |
| Scheduled tasks removed | Amazon EC2 Launch tasks |
| Driver files removed | `xen*.sys` |
| UpperFilters cleaned | Remove `XENFILT` from System and HDC class GUIDs |

### EC2 Launch

**Remove:** [`../../pkg/convert-windows/hypervisor/plugins/remove/ec2launch/remove.go`](../../pkg/convert-windows/hypervisor/plugins/remove/ec2launch/remove.go)

| Action | Details |
|--------|---------|
| Registry keys removed | EC2Launch/Ec2Config uninstall keys |
| Directories removed | `Program Files\Amazon\EC2Launch`, `Program Files\Amazon\Ec2ConfigService` |

### Nutanix

**Remove:** [`../../pkg/convert-windows/hypervisor/plugins/remove/nutanix/remove.go`](../../pkg/convert-windows/hypervisor/plugins/remove/nutanix/remove.go)
**Services:** [`../../pkg/convert-windows/hypervisor/plugins/services/nutanix/services.go`](../../pkg/convert-windows/hypervisor/plugins/services/nutanix/services.go)

**Software removal:**

| Action | Details |
|--------|---------|
| Directories removed | `Program Files\Nutanix`, `Program Files (x86)\Nutanix` |
| Registry keys removed | `Uninstall\Nutanix Guest Tools` |

**Services disabled (Start=4):**
`NutanixGuestTools`, `NutanixGuestAgent`, `NgtService`

### VirtualBox

**Remove:** [`../../pkg/convert-windows/hypervisor/plugins/remove/virtualbox/remove.go`](../../pkg/convert-windows/hypervisor/plugins/remove/virtualbox/remove.go)
**Services:** [`../../pkg/convert-windows/hypervisor/plugins/services/virtualbox/services.go`](../../pkg/convert-windows/hypervisor/plugins/services/virtualbox/services.go)

**Software removal:**

| Action | Details |
|--------|---------|
| Directories removed | `Program Files\Oracle\VirtualBox Guest Additions` |
| Registry keys removed | `Uninstall\Oracle VM VirtualBox Guest Additions` |
| Driver files removed | All `vbox*.sys` from `System32\drivers` |

**Services disabled (Start=4):**
`VBoxService`, `VBoxGuest`, `VBoxSF`, `VBoxVideo`, `VBoxMouse`

### Hyper-V

**Remove:** [`../../pkg/convert-windows/hypervisor/plugins/remove/hyperv/remove.go`](../../pkg/convert-windows/hypervisor/plugins/remove/hyperv/remove.go)
**Services:** [`../../pkg/convert-windows/hypervisor/plugins/services/hyperv/services.go`](../../pkg/convert-windows/hypervisor/plugins/services/hyperv/services.go)

Hyper-V integration components are inbox Windows drivers and cannot be safely
removed offline. The remove plugin logs their presence; the services plugin
disables them.

**Services disabled (Start=4):**
`vmicheartbeat`, `vmicshutdown`, `vmicvss`, `vmictimesync`, `vmicrdv`,
`vmicguestinterface`, `vmickvpexchange`, `vmicvmsession`, `storflt`

**Extra registry:** `W32Time\TimeProviders\VMICTimeProvider` `Enabled=0`

**Inbox drivers left in place:**
`vmbus.sys`, `storvsc.sys`, `netvsc.sys`, `VMBusHID.sys`, `hypervideo.sys`

### Citrix/XenServer

**Remove:** [`../../pkg/convert-windows/hypervisor/plugins/remove/citrix/remove.go`](../../pkg/convert-windows/hypervisor/plugins/remove/citrix/remove.go)

| Action | Details |
|--------|---------|
| UpperFilters cleaned | Remove `XENFILT` from System and HDC class GUIDs |
| Services disabled (Start=4) | `XenSvc`, `xenagent`, `xenbus_monitor`, `xenlite` |
| Directories removed | `Program Files\Citrix\XenTools` |
| Registry keys removed | `Uninstall\Citrix XenTools` |

Boot-critical filter cleanup only; full MSI/driver scrub is not implemented.

### Parallels

**Remove:** [`../../pkg/convert-windows/hypervisor/plugins/remove/parallels/remove.go`](../../pkg/convert-windows/hypervisor/plugins/remove/parallels/remove.go)

| Action | Details |
|--------|---------|
| LowerFilters cleaned | Remove `prl_strg` from disk class GUID (prevents BSOD 0x7b) |
| Services disabled (Start=4) | `prl_strg`, `prl_boot`, `prl_scsi`, `prl_eth5`, `Parallels Tools Service` |
| Directories removed | `Program Files\Parallels\Parallels Tools`, `Program Files (x86)\Parallels\Parallels Tools` |
| Registry keys removed | `Uninstall\Parallels Tools` |

Boot-critical filter + core service disable; a full leftover-driver scrub is not implemented.

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
| Citrix | Yes (filter + core services) | Inline in remove | Partial |
| Parallels | Yes (filter + core services) | Inline in remove | Partial |

---

## Windows: Install (virtio drivers + guest agent)

Driver and guest agent installation depends on the Windows version, not the
source hypervisor.

**Version handler interface:** [`pkg/convert-windows/version/version.go`](../../pkg/convert-windows/version/version.go)
**Version handlers:** [`pkg/convert-windows/version/handlers.go`](../../pkg/convert-windows/version/handlers.go)
**Version registration:** [`pkg/convert-windows/version/register.go`](../../pkg/convert-windows/version/register.go)

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
- driverdb (Win8+): [`pkg/convert-windows/drivers/plugins/driverdb/`](../../pkg/convert-windows/drivers/plugins/driverdb/)
- criticaldb (Win7 and older): [`pkg/convert-windows/drivers/plugins/criticaldb/`](../../pkg/convert-windows/drivers/plugins/criticaldb/)

**Guest agent exclusions:** [`pkg/convert-windows/version/guestagent.go`](../../pkg/convert-windows/version/guestagent.go)

### What gets installed

**Driver copy:** [`pkg/convert-windows/drivers/copy.go`](../../pkg/convert-windows/drivers/copy.go)
**Driver source/collection:** [`pkg/convert-windows/driversource/`](../../pkg/convert-windows/driversource/)
**DevicePath update:** [`pkg/convert-windows/drivers/devicepath.go`](../../pkg/convert-windows/drivers/devicepath.go)
**Crash control:** [`pkg/convert-windows/crashcontrol/`](../../pkg/convert-windows/crashcontrol/)

| Component | Details |
|-----------|---------|
| VirtIO drivers copied | `.sys`, `.inf`, `.cat` files to `C:\Windows\Drivers\VirtIO\` |
| Boot-critical drivers | `viostor.sys`, `vioscsi.sys` also copied to `System32\drivers\` |
| Driver registration | Service entries with `Start=0` (boot start) for `viostor`/`vioscsi` |
| DevicePath | `%SystemRoot%\Drivers\VirtIO` appended to registry `DevicePath` |
| Crash control | Disables auto-reboot on BSOD (`AutoReboot=0`) |
| NTFS heads fix | Win2003 and WinXP only |

### Firstboot install scripts

**Firstboot framework:** [`pkg/convert-windows/firstboot/`](../../pkg/convert-windows/firstboot/)
**Firstboot plugins:** [`pkg/convert-windows/firstboot/plugins/`](../../pkg/convert-windows/firstboot/plugins/)

| Priority | Component | Description |
|----------|-----------|-------------|
| 2000 | [pnputil](../../pkg/convert-windows/firstboot/plugins/pnputil/) | Install VirtIO drivers via PnP |
| 2500 | [staticip](../../pkg/convert-windows/firstboot/plugins/staticipfb/) | Reconfigure static IPs (method varies by OS version) |
| 2600 | [routecleanup](../../pkg/convert-windows/firstboot/plugins/routecleanup/) | Remove duplicate persistent routes |
| 2700 | [multipleips](../../pkg/convert-windows/firstboot/plugins/multipleips/) | Add secondary IPs per NIC |
| 3000 | [qemuga](../../pkg/convert-windows/firstboot/plugins/qemuga/) | Install QEMU guest agent MSI (Win7+ only) |
| 4000 | [diskonliner](../../pkg/convert-windows/firstboot/plugins/diskonliner/) | Bring offline disks online |
| 9100 | [vmwarecleanup](../../pkg/convert-windows/firstboot/plugins/vmwarecleanup/) | Remove residual VMware drivers/services |
| 99999 | [signal](../../pkg/convert-windows/firstboot/plugins/signal/) | Signal conversion completion on COM1 (last script; launcher then cleans up and reboots) |

After scripts finish, `firstboot.bat` schedules
`C:\Windows\System32\shutdown.exe /r /t 5 /f` then removes `Guestfs\Firstboot`
so reboot-required installers can complete. With `WaitForGuestReboot`, COM1
`CONVERSION_DONE` is sent before that reboot (Forklift wait-for-reboot order).

## Cross-Reference Matrices

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
| Citrix | Partial (XENFILT + core services) |
| Parallels | Partial (prl_strg filter + core services) |

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

## Gaps and Notes

1. **Windows Citrix/Parallels cleanup** -- Boot-critical filter and core
   service cleanup is implemented; full MSI/driver scrub is still partial.
   Install is unaffected (VirtIO drivers install normally regardless of
   source hypervisor).

2. **Guest agent exclusions** -- `win2008`, `winvista`, `win2003`, and `winxp`
   do not receive the QEMU guest agent MSI (see
   [`pkg/convert-windows/version/guestagent.go`](../../pkg/convert-windows/version/guestagent.go)).
   VirtIO driver installation works normally for these versions.

---

## Related docs

- [conversion-paths.md](conversion-paths.md) — overview (cleanup vs install)
- [conversion-paths-linux.md](conversion-paths-linux.md) — Linux paths
- [guest-os-handlers.md](guest-os-handlers.md) — version handler classification
- [../apps/kc-convert-windows.md](../apps/kc-convert-windows.md) — Windows converter pipeline blocks
