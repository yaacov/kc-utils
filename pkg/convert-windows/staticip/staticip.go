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
			newIPCmd += fmt.Sprintf(" -PrefixLength %s", netmaskToPrefixLength(sip.Netmask))
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
func RegistryScript(ips []types.StaticIP) string {
	if len(ips) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("# Registry-based static IP configuration\r\n")
	for i, sip := range ips {
		mac := strings.ToUpper(strings.ReplaceAll(sip.MAC, ":", "-"))
		key := fmt.Sprintf("HKLM:\\SYSTEM\\CurrentControlSet\\Services\\Tcpip\\Parameters\\Interfaces\\{%s}", mac)
		b.WriteString(fmt.Sprintf("$key%d = '%s'\r\n", i, key))
		b.WriteString(fmt.Sprintf("New-Item -Path $key%d -Force | Out-Null\r\n", i))
		b.WriteString(fmt.Sprintf("Set-ItemProperty -Path $key%d -Name EnableDHCP -Value 0\r\n", i))
		b.WriteString(fmt.Sprintf("Set-ItemProperty -Path $key%d -Name IPAddress -Value @('%s')\r\n", i, sip.IP))
		if sip.Netmask != "" {
			b.WriteString(fmt.Sprintf("Set-ItemProperty -Path $key%d -Name SubnetMask -Value @('%s')\r\n", i, sip.Netmask))
		}
		if sip.Gateway != "" {
			b.WriteString(fmt.Sprintf("Set-ItemProperty -Path $key%d -Name DefaultGateway -Value @('%s')\r\n", i, sip.Gateway))
		}
		if len(sip.DNS) > 0 {
			b.WriteString(fmt.Sprintf("Set-ItemProperty -Path $key%d -Name NameServer -Value '%s'\r\n", i, strings.Join(sip.DNS, ",")))
		}
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
		mac := strings.ToUpper(strings.ReplaceAll(sip.MAC, ":", "-"))
		b.WriteString(fmt.Sprintf("$cfg = Get-WmiObject Win32_NetworkAdapterConfiguration | Where-Object { $_.MACAddress -eq %q }\r\n", mac))
		b.WriteString("if ($cfg) {\r\n")
		if sip.IP != "" && sip.Netmask != "" {
			b.WriteString(fmt.Sprintf("    $cfg.EnableStatic(@(%q), @(%q)) | Out-Null\r\n", sip.IP, sip.Netmask))
		}
		if sip.Gateway != "" {
			b.WriteString(fmt.Sprintf("    $cfg.SetGateways(@(%q)) | Out-Null\r\n", sip.Gateway))
		}
		if len(sip.DNS) > 0 {
			b.WriteString(fmt.Sprintf("    $cfg.SetDNSServerSearchOrder(@(%s)) | Out-Null\r\n", quoteList(sip.DNS)))
			b.WriteString(fmt.Sprintf("    netsh interface ip set dns name=\"\" static %s primary validate=no\r\n", sip.DNS[0]))
		}
		b.WriteString("}\r\n")
	}
	return b.String()
}

// RegistryBatScript generates batch reg.exe static IP configuration for guests without PowerShell.
func RegistryBatScript(ips []types.StaticIP) string {
	if len(ips) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("@echo off\r\n")
	b.WriteString("REM Registry-based static IP configuration\r\n")
	for _, sip := range ips {
		mac := strings.ToUpper(strings.ReplaceAll(sip.MAC, ":", "-"))
		key := fmt.Sprintf(`HKLM\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces\{%s}`, mac)
		b.WriteString(fmt.Sprintf("reg add %q /v EnableDHCP /t REG_DWORD /d 0 /f\r\n", key))
		b.WriteString(fmt.Sprintf("reg add %q /v IPAddress /t REG_MULTI_SZ /d %s /f\r\n", key, sip.IP))
		if sip.Netmask != "" {
			b.WriteString(fmt.Sprintf("reg add %q /v SubnetMask /t REG_MULTI_SZ /d %s /f\r\n", key, sip.Netmask))
		}
		if sip.Gateway != "" {
			b.WriteString(fmt.Sprintf("reg add %q /v DefaultGateway /t REG_MULTI_SZ /d %s /f\r\n", key, sip.Gateway))
		}
		if len(sip.DNS) > 0 {
			b.WriteString(fmt.Sprintf("reg add %q /v NameServer /t REG_SZ /d %s /f\r\n", key, strings.Join(sip.DNS, ",")))
		}
	}
	return b.String()
}

// DevconVMwareCleanupBat returns a batch script to remove VMware devices via devcon.
func DevconVMwareCleanupBat() string {
	return "@echo off\r\n" +
		"REM Remove VMware drivers and services\r\n" +
		"for %%S in (VMTools VGAuthService VMware Physical Disk Helper VMware Tools) do (\r\n" +
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

func netmaskToPrefixLength(mask string) string {
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
