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

// RebootSignalScript returns the COM1 conversion-done signal script.
func RebootSignalScript() string {
	return "# Signal conversion completion on COM1\r\n" +
		"cmd /c \"echo CONVERSION_DONE>\\\\.\\COM1\" 2>&1 | Out-Null\r\n"
}

// VMwareCleanupScript returns a PowerShell script for VMware driver and service removal.
func VMwareCleanupScript() string {
	return "# Remove VMware drivers and services\r\n" +
		"Get-PnpDevice | Where-Object { $_.FriendlyName -like '*VMware*' } | Disable-PnpDevice -Confirm:$false -ErrorAction SilentlyContinue\r\n" +
		"Get-Service | Where-Object { $_.Name -like '*VMware*' -or $_.Name -like '*VMTools*' -or $_.Name -like '*VGAuth*' } | Stop-Service -Force -ErrorAction SilentlyContinue\r\n" +
		"Get-Service | Where-Object { $_.Name -like '*VMware*' -or $_.Name -like '*VMTools*' -or $_.Name -like '*VGAuth*' } | ForEach-Object { sc.exe delete $_.Name } 2>&1 | Out-Null\r\n"
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
