package staticip

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

// PowerShellScript generates a firstboot PowerShell script for static IP configuration.
func PowerShellScript(ips []types.StaticIP) string {
	if len(ips) == 0 {
		return ""
	}
	var lines []string
	lines = append(lines, "# Configure static IP addresses")
	for _, sip := range ips {
		macNorm := strings.ReplaceAll(sip.MAC, ":", "")
		adapterVar := "adapter_" + macNorm
		lines = append(lines,
			fmt.Sprintf(`$%s = Get-NetAdapter | Where-Object { ($_.MacAddress -replace '-','') -eq %q }`, adapterVar, macNorm),
			fmt.Sprintf("if ($%s) {", adapterVar),
		)

		newIPCmd := fmt.Sprintf(`    New-NetIPAddress -InterfaceIndex $%s.InterfaceIndex -IPAddress %q`, adapterVar, sip.IP)
		if sip.Netmask != "" {
			newIPCmd += fmt.Sprintf(" -PrefixLength %s", NetmaskToPrefixLength(sip.Netmask))
		}
		if sip.Gateway != "" {
			newIPCmd += fmt.Sprintf(` -DefaultGateway %q`, sip.Gateway)
		}
		lines = append(lines, newIPCmd)

		if len(sip.DNS) > 0 {
			quoted := make([]string, len(sip.DNS))
			for i, d := range sip.DNS {
				quoted[i] = `"` + d + `"`
			}
			lines = append(lines, fmt.Sprintf(`    Set-DnsClientServerAddress -InterfaceIndex $%s.InterfaceIndex -ServerAddresses %s`,
				adapterVar, strings.Join(quoted, ",")))
		}
		lines = append(lines, "}")
	}
	return strings.Join(lines, "\r\n") + "\r\n"
}

// RegistryScript generates registry-based static IP configuration for Windows guests.
//
// The per-NIC settings live under
// ...\Tcpip\Parameters\Interfaces\<interface-GUID>, keyed by the adapter's
// interface GUID (a value Windows assigns) — never by the MAC address. That GUID
// is unknown offline, so the script resolves it at first boot. This path targets
// Windows 7 / Server 2008 R2, which ship PowerShell 2.0 but not the Get-NetAdapter
// cmdlet, so it uses WMI (Win32_NetworkAdapterConfiguration.SettingID is the
// interface GUID) to find the adapter by MAC.
func RegistryScript(ips []types.StaticIP) string {
	if len(ips) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("# Registry-based static IP configuration\r\n")
	for i, sip := range ips {
		macNorm := normalizeMAC(sip.MAC)
		b.WriteString(fmt.Sprintf("$cfg%d = Get-WmiObject Win32_NetworkAdapterConfiguration | Where-Object { ($_.MACAddress -replace '[^0-9A-Fa-f]','') -eq %q }\r\n", i, macNorm))
		b.WriteString(fmt.Sprintf("if ($cfg%d -and $cfg%d.SettingID) {\r\n", i, i))
		b.WriteString(fmt.Sprintf("    $key%d = \"HKLM:\\SYSTEM\\CurrentControlSet\\Services\\Tcpip\\Parameters\\Interfaces\\$($cfg%d.SettingID)\"\r\n", i, i))
		b.WriteString(fmt.Sprintf("    New-Item -Path $key%d -Force | Out-Null\r\n", i))
		b.WriteString(fmt.Sprintf("    Set-ItemProperty -Path $key%d -Name EnableDHCP -Value 0\r\n", i))
		b.WriteString(fmt.Sprintf("    Set-ItemProperty -Path $key%d -Name IPAddress -Value @('%s')\r\n", i, sip.IP))
		if sip.Netmask != "" {
			b.WriteString(fmt.Sprintf("    Set-ItemProperty -Path $key%d -Name SubnetMask -Value @('%s')\r\n", i, sip.Netmask))
		}
		if sip.Gateway != "" {
			b.WriteString(fmt.Sprintf("    Set-ItemProperty -Path $key%d -Name DefaultGateway -Value @('%s')\r\n", i, sip.Gateway))
		}
		if len(sip.DNS) > 0 {
			b.WriteString(fmt.Sprintf("    Set-ItemProperty -Path $key%d -Name NameServer -Value '%s'\r\n", i, strings.Join(sip.DNS, ",")))
		}
		b.WriteString("}\r\n")
	}
	return b.String()
}

// RebootSignalScript returns the PowerShell COM1 conversion-done signal script.
func RebootSignalScript() string {
	return "# Signal conversion completion on COM1\r\n" +
		"cmd /c \"echo CONVERSION_DONE>\\\\.\\COM1\" 2>&1 | Out-Null\r\n"
}

// RebootSignalBatScript returns the batch COM1 conversion-done signal script
// for guests without PowerShell (XP / Server 2003).
func RebootSignalBatScript() string {
	return "@echo off\r\n" +
		"REM Signal conversion completion on COM1\r\n" +
		"echo CONVERSION_DONE>\\\\.\\COM1\r\n"
}

// VMwareCleanupScript returns a PowerShell script for VMware driver and service removal.
func VMwareCleanupScript() string {
	return "# Remove VMware drivers and services\r\n" +
		"Get-PnpDevice | Where-Object { $_.FriendlyName -like '*VMware*' } | Disable-PnpDevice -Confirm:$false -ErrorAction SilentlyContinue\r\n" +
		"Get-Service | Where-Object { $_.Name -like '*VMware*' -or $_.Name -like '*VMTools*' -or $_.Name -like '*VGAuth*' } | Stop-Service -Force -ErrorAction SilentlyContinue\r\n" +
		"Get-Service | Where-Object { $_.Name -like '*VMware*' -or $_.Name -like '*VMTools*' -or $_.Name -like '*VGAuth*' } | ForEach-Object { sc.exe delete $_.Name } 2>&1 | Out-Null\r\n"
}

// WMIScript generates WMI + netsh static IP configuration for legacy Windows.
func WMIScript(ips []types.StaticIP) string {
	if len(ips) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("# Configure static IP via WMI and netsh\r\n")
	for _, sip := range ips {
		// WMI reports MACAddress as colon-separated hex; strip all separators on
		// both sides so the match is independent of formatting.
		macNorm := normalizeMAC(sip.MAC)
		b.WriteString(fmt.Sprintf("$cfg = Get-WmiObject Win32_NetworkAdapterConfiguration | Where-Object { ($_.MACAddress -replace '[^0-9A-Fa-f]','') -eq %q }\r\n", macNorm))
		b.WriteString("if ($cfg) {\r\n")
		if sip.IP != "" && sip.Netmask != "" {
			b.WriteString(fmt.Sprintf("    $cfg.EnableStatic(@(%q), @(%q)) | Out-Null\r\n", sip.IP, sip.Netmask))
		}
		if sip.Gateway != "" {
			b.WriteString(fmt.Sprintf("    $cfg.SetGateways(@(%q)) | Out-Null\r\n", sip.Gateway))
		}
		if len(sip.DNS) > 0 {
			// SetDNSServerSearchOrder already applies the full DNS list for this
			// adapter. A netsh fallback would need the interface name (not the
			// MAC), which we do not have here, so there is nothing to add.
			b.WriteString(fmt.Sprintf("    $cfg.SetDNSServerSearchOrder(@(%s)) | Out-Null\r\n", quoteList(sip.DNS)))
		}
		b.WriteString("}\r\n")
	}
	return b.String()
}

// RegistryBatScript generates batch reg.exe static IP configuration for guests
// without PowerShell (Windows XP / Server 2003).
//
// Like RegistryScript, the Interfaces subkey is the adapter's interface GUID, not
// the MAC. With no PowerShell available, the GUID is resolved at first boot with
// wmic (Win32_NetworkAdapterConfiguration.SettingID). Piping wmic through findstr
// normalizes its trailing-CR line endings, and delayed expansion lets the
// resolved GUID be used later in the loop.
func RegistryBatScript(ips []types.StaticIP) string {
	if len(ips) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("@echo off\r\n")
	b.WriteString("REM Registry-based static IP configuration\r\n")
	b.WriteString("setlocal enabledelayedexpansion\r\n")
	for i, sip := range ips {
		mac := formatMACColon(sip.MAC)
		b.WriteString(fmt.Sprintf("set \"GUID%d=\"\r\n", i))
		b.WriteString(fmt.Sprintf("for /f \"usebackq tokens=2 delims==\" %%%%G in (`wmic nicconfig where \"MACAddress='%s'\" get SettingID /value 2^>nul ^| findstr /i \"SettingID=\"`) do set \"GUID%d=%%%%G\"\r\n", mac, i))
		b.WriteString(fmt.Sprintf("if defined GUID%d (\r\n", i))
		// Quotes are part of the key string (not %q, which would double the
		// backslashes in the registry path) so reg.exe sees single backslashes.
		key := fmt.Sprintf(`"HKLM\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces\!GUID%d!"`, i)
		b.WriteString(fmt.Sprintf("    reg add %s /v EnableDHCP /t REG_DWORD /d 0 /f\r\n", key))
		b.WriteString(fmt.Sprintf("    reg add %s /v IPAddress /t REG_MULTI_SZ /d %s /f\r\n", key, sip.IP))
		if sip.Netmask != "" {
			b.WriteString(fmt.Sprintf("    reg add %s /v SubnetMask /t REG_MULTI_SZ /d %s /f\r\n", key, sip.Netmask))
		}
		if sip.Gateway != "" {
			b.WriteString(fmt.Sprintf("    reg add %s /v DefaultGateway /t REG_MULTI_SZ /d %s /f\r\n", key, sip.Gateway))
		}
		if len(sip.DNS) > 0 {
			b.WriteString(fmt.Sprintf("    reg add %s /v NameServer /t REG_SZ /d %s /f\r\n", key, strings.Join(sip.DNS, ",")))
		}
		b.WriteString(")\r\n")
	}
	b.WriteString("endlocal\r\n")
	return b.String()
}

// normalizeMAC returns the MAC as uppercase hex with every separator removed, so
// it can be compared against WMI or Get-NetAdapter values regardless of whether
// they use ':' or '-'.
func normalizeMAC(mac string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(mac) {
		if (r >= '0' && r <= '9') || (r >= 'A' && r <= 'F') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// formatMACColon returns the MAC as colon-separated uppercase hex
// (e.g. "52:54:00:AA:BB:CC"), the format WMI expects in a query.
func formatMACColon(mac string) string {
	norm := normalizeMAC(mac)
	var parts []string
	for i := 0; i+2 <= len(norm); i += 2 {
		parts = append(parts, norm[i:i+2])
	}
	return strings.Join(parts, ":")
}

// DevconVMwareCleanupBat returns a batch script to remove VMware devices via devcon.
func DevconVMwareCleanupBat() string {
	return "@echo off\r\n" +
		"REM Remove VMware drivers and services\r\n" +
		// Batch `for` splits its set on whitespace, so each name must be a single
		// token. These are the real service names (no display names with spaces),
		// matching the VMware service disabler list.
		"for %%S in (VMTools VGAuthService VMwarePhysicalDiskHelper vm3dservice VMUSBArbService) do (\r\n" +
		"    sc stop \"%%S\" 2>nul\r\n" +
		"    sc delete \"%%S\" 2>nul\r\n" +
		")\r\n" +
		"if exist \"C:\\Windows\\System32\\drivers\\vm3dmp.sys\" del /f /q \"C:\\Windows\\System32\\drivers\\vm3dmp.sys\" 2>nul\r\n"
}

func quoteList(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = fmt.Sprintf("%q", v)
	}
	return strings.Join(quoted, ",")
}

// NetmaskToPrefixLength converts a dotted IPv4 netmask to a CIDR prefix length string.
// Invalid masks default to "24".
func NetmaskToPrefixLength(mask string) string {
	ip := net.ParseIP(mask)
	if ip == nil {
		return "24"
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return "24"
	}
	ones, bits := net.IPMask(ip4).Size()
	if bits == 0 {
		return "24"
	}
	return strconv.Itoa(ones)
}
