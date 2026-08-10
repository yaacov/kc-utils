# firstboot plugins (Windows)

`Contributor` interface — generate PowerShell or batch firstboot scripts for Windows guests.

Each contributor generates a script fragment (PowerShell or batch) that runs
during the guest's first boot under KVM. The firstboot orchestrator collects
all contributors whose `ShouldRun` returns true, sorts them by `Priority`
(lower runs first), writes each script to
`Program Files\Guestfs\Firstboot\scripts\`, and registers a `RunOnce` registry
entry that launches them via `firstboot.bat`. After all scripts complete, the
launcher schedules a guest reboot
(`C:\Windows\System32\shutdown.exe /r /t 5 /f`) then cleans up the Firstboot
directory so installers that returned “reboot required” can finish. Contributors declare whether they need PowerShell (`UsesBatch` returns
false) or batch (`UsesBatch` returns true) to support older Windows versions
that lack PowerShell.

| Key | Package | Priority | Contributor |
|-----|---------|----------|-------------|
| `pnputil` | pnputil/ | 2000 | Install staged VirtIO drivers via pnputil |
| `staticipfb` | staticipfb/ | 2500 | Apply static IP configuration |
| `routecleanup` | routecleanup/ | 2600 | Clean stale network routes |
| `multipleips` | multipleips/ | 2700 | Configure additional IP addresses |
| `qemuga` | qemuga/ | 3000 | Install QEMU guest agent |
| `diskonliner` | diskonliner/ | 4000 | Bring offline disks online at first boot |
| `vmwarecleanup` | vmwarecleanup/ | 9100 | Remove VMware Tools leftovers |
| `signal` | signal/ | 99999 | Signal conversion completion to orchestrator |

## pnputil

**What it does:** Installs the staged VirtIO drivers using the Windows
`pnputil` command so non-boot devices (network, display, balloon) are
recognized.

**How it works:** Generates a `pnputil /add-driver ... /install` command for
each `.inf` file in the `Windows\Drivers\VirtIO\` directory. Runs early
(priority 2000) so drivers are available before network configuration scripts.
Supports both PowerShell and batch output for old Windows versions.

## staticipfb

**What it does:** Configures static IP addresses on the guest's NICs after
the virtio network driver is loaded.

**How it works:** Generates a PowerShell script that waits up to 5 minutes for
the `netkvm.sys` VirtIO network driver to appear, then applies IP addresses
from the StaticIP list. Supports three modes depending on Windows version:
`New-NetIPAddress` cmdlets (modern), WMI/netsh (older), or registry-based IP
configuration (oldest). Can fall back to batch scripts for pre-PowerShell
Windows.

## routecleanup

**What it does:** Removes duplicate and stale persistent network routes that
accumulate when a NIC changes hardware (old routes reference dead interfaces).

**How it works:** Generates a PowerShell script that enumerates persistent
routes, identifies duplicates (same destination and gateway on different
interfaces), keeps the route on the live adapter with the lowest metric for
default gateway routes, and removes stale routes on dead interfaces via
`Remove-NetRoute` with `route.exe` fallback. In registry mode, also cleans the
`PersistentRoutes` registry key.

## multipleips

**What it does:** Adds secondary (complementary) IP addresses to NICs that
have more than one IP configured on the source VM.

**How it works:** Groups StaticIP entries by MAC address. For each MAC with
multiple IPs, generates a PowerShell script that waits for the VirtIO network
driver, finds the adapter by MAC, and adds secondary IPs via
`New-NetIPAddress` (skipping the first IP, which is set by `staticipfb`).
Sets DNS servers from all IPs on the NIC. Registry mode writes all IPs as
multi-string values into the interface registry key.

## qemuga

**What it does:** Installs the QEMU Guest Agent MSI package so the KubeVirt
host can communicate with the guest for graceful shutdown, filesystem freeze,
and monitoring.

**How it works:** Generates a PowerShell script that installs the exact
`qemu-ga*.msi` basename from the filtered `DriverFiles` list
(`C:\Windows\Drivers\VirtIO\<msi>`) via `msiexec /i ... /qn /norestart`.
Only runs when a guest agent MSI was selected and passed completeness checks.

## diskonliner

**What it does:** Brings all offline disks online on first boot so data disks
are accessible after conversion.

**How it works:** Two modes depending on Windows version. Modern Windows uses
PowerShell `Get-Disk | Where-Object IsOffline | Set-Disk -IsOffline $false`
with ReadOnly cleared. Older Windows uses a WMI query to enumerate disks and
`diskpart.exe` to online each one.

## vmwarecleanup

**What it does:** Removes residual VMware Tools drivers and services that the
offline removal step could not fully clean (some components require Windows to
be running).

**How it works:** Generates a PowerShell script that uses `sc.exe delete` to
remove VMware services. Only runs when `Options.VMwareDriverRemoval` is true.
Supports a batch mode (`devcon.exe`) for older Windows versions.

## signal

**What it does:** Signals the orchestrator that conversion is complete and the
guest has successfully booted under KVM (same COM1 marker Forklift injects for
virt-v2v).

**How it works:** Generates a script that writes `CONVERSION_DONE` to COM1.
Modern guests get a PowerShell script (`cmd /c "echo CONVERSION_DONE>\\.\COM1"`);
XP/Server 2003 (`UsesBatch` when PowerShell is unsupported) get a `.bat` that
writes the same marker with `echo CONVERSION_DONE>\\.\COM1`. Runs last among
scripts (priority 99999) so all other firstboot work is complete before
signaling. Only active when `Options.WaitForGuestReboot` is true. The launcher
then cleans up and reboots; Forklift's wait-for-reboot watcher expects that
COM1-then-reboot order.
