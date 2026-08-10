# staticip -- PowerShell/registry static IP script generation

Generates firstboot scripts that configure static IP addresses on converted Windows guests. Multiple script formats are provided to cover the full range of Windows versions, from modern PowerShell cmdlets to legacy WMI/netsh commands to raw `reg.exe` batch scripts for guests without PowerShell.

Each generator takes a slice of `StaticIP` entries (MAC, IP, netmask, gateway, DNS) and emits a complete script. `PowerShellScript` uses `New-NetIPAddress` and `Set-DnsClientServerAddress` cmdlets for Windows 8+. `RegistryScript` writes Tcpip interface parameters via PowerShell registry commands for Windows 7/2008 R2. `WMIScript` uses WMI `Win32_NetworkAdapterConfiguration` and netsh for Vista/Server 2008. `RegistryBatScript` uses `reg.exe` for pre-PowerShell guests (XP, Server 2003). The package also provides VMware cleanup scripts and a COM1 conversion-done signal script.

## Key exports

| Symbol | Role |
|--------|------|
| `PowerShellScript` | Generates a PowerShell script using `New-NetIPAddress`/`Set-DnsClientServerAddress` cmdlets |
| `RegistryScript` | Generates a PowerShell script that writes static IP settings via registry keys |
| `WMIScript` | Generates a PowerShell script using WMI and netsh for legacy Windows |
| `RegistryBatScript` | Generates a batch script using `reg.exe` for guests without PowerShell |
| `RebootSignalScript` | Returns a PowerShell script that signals conversion completion on COM1 |
| `RebootSignalBatScript` | Returns a batch script that signals conversion completion on COM1 (XP/2003) |
| `VMwareCleanupScript` | Returns a PowerShell script to remove VMware drivers and services |
| `DevconVMwareCleanupBat` | Returns a batch script to remove VMware devices and services via `sc` |
