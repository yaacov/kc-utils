package multipleips

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"
)

type Plugin struct{}

func init() {
	firstboot.Contributors.Register("multipleips", &Plugin{})
}

func (p *Plugin) Priority() int { return 2700 }
func (p *Plugin) Name() string  { return "preserve-complementary-ips" }

func (p *Plugin) ShouldRun(cfg *firstboot.ContributorConfig) bool {
	if cfg.Offline || !cfg.Options.MultipleIPsPerNic || len(cfg.StaticIPs) == 0 {
		return false
	}
	if cfg.Version != nil && !cfg.Version.SupportsPowerShell() {
		return false
	}
	// Only run if there are multiple IPs for at least one MAC
	return hasMultipleIPsPerMAC(cfg.StaticIPs)
}

func (p *Plugin) UsesBatch(_ *firstboot.ContributorConfig) bool { return false }

func (p *Plugin) Generate(cfg *firstboot.ContributorConfig) (string, error) {
	if cfg.Options.WindowsRegistryNetwork {
		return registryScript(cfg.StaticIPs), nil
	}
	return powershellScript(cfg.StaticIPs), nil
}

type nicConfig struct {
	MAC string
	IPs []types.StaticIP
}

func hasMultipleIPsPerMAC(ips []types.StaticIP) bool {
	counts := make(map[string]int)
	for _, sip := range ips {
		counts[strings.ToLower(sip.MAC)]++
	}
	for _, c := range counts {
		if c > 1 {
			return true
		}
	}
	return false
}

func groupByMAC(ips []types.StaticIP) []nicConfig {
	order := make([]string, 0)
	grouped := make(map[string]*nicConfig)
	for _, sip := range ips {
		macLower := strings.ToLower(sip.MAC)
		if _, ok := grouped[macLower]; !ok {
			order = append(order, macLower)
			grouped[macLower] = &nicConfig{MAC: sip.MAC}
		}
		grouped[macLower].IPs = append(grouped[macLower].IPs, sip)
	}
	result := make([]nicConfig, 0, len(order))
	for _, mac := range order {
		cfg := grouped[mac]
		if len(cfg.IPs) > 1 {
			result = append(result, *cfg)
		}
	}
	return result
}

func powershellScript(ips []types.StaticIP) string {
	configs := groupByMAC(ips)
	if len(configs) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("# Add complementary (secondary) IPs to NICs\r\n")
	b.WriteString("$deadline = (Get-Date).AddMinutes(5)\r\n")
	b.WriteString("while ((Get-Date) -lt $deadline) {\r\n")
	b.WriteString("    if (Get-NetAdapter -Physical | Where-Object DriverFileName -eq 'netkvm.sys') { break }\r\n")
	b.WriteString("    Start-Sleep -Seconds 5\r\n")
	b.WriteString("}\r\n\r\n")

	for _, cfg := range configs {
		macNorm := strings.ReplaceAll(cfg.MAC, ":", "")
		varName := "adapter_" + macNorm
		b.WriteString(fmt.Sprintf("$%s = Get-NetAdapter -Physical | Where-Object { ($_.MacAddress -replace '-','') -eq %q }\r\n",
			varName, strings.ToUpper(macNorm)))
		b.WriteString(fmt.Sprintf("if ($%s) {\r\n", varName))

		// Skip the first IP (already set by static-ip plugin), add secondary IPs
		for i := 1; i < len(cfg.IPs); i++ {
			sip := cfg.IPs[i]
			cmd := fmt.Sprintf("    New-NetIPAddress -InterfaceIndex $%s.InterfaceIndex -IPAddress %q",
				varName, sip.IP)
			if sip.Netmask != "" {
				cmd += fmt.Sprintf(" -PrefixLength %s", netmaskToPrefix(sip.Netmask))
			}
			// Secondary IPs should not set gateway
			b.WriteString(cmd + "\r\n")
		}

		// Set DNS once with all servers from all IPs for this NIC
		allDNS := collectDNS(cfg.IPs)
		if len(allDNS) > 0 {
			quoted := make([]string, len(allDNS))
			for i, d := range allDNS {
				quoted[i] = `"` + d + `"`
			}
			b.WriteString(fmt.Sprintf("    Set-DnsClientServerAddress -InterfaceIndex $%s.InterfaceIndex -ServerAddresses %s\r\n",
				varName, strings.Join(quoted, ",")))
		}
		b.WriteString("}\r\n\r\n")
	}
	return b.String()
}

func registryScript(ips []types.StaticIP) string {
	configs := groupByMAC(ips)
	if len(configs) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("# Add complementary IPs via registry\r\n")
	b.WriteString("$deadline = (Get-Date).AddMinutes(5)\r\n")
	b.WriteString("while ((Get-Date) -lt $deadline) {\r\n")
	b.WriteString("    if (Get-NetAdapter -Physical | Where-Object DriverFileName -eq 'netkvm.sys') { break }\r\n")
	b.WriteString("    Start-Sleep -Seconds 5\r\n")
	b.WriteString("}\r\n\r\n")

	for i, cfg := range configs {
		macNorm := strings.ReplaceAll(cfg.MAC, ":", "")
		varName := fmt.Sprintf("adapter_%s", macNorm)

		b.WriteString(fmt.Sprintf("$%s = Get-NetAdapter -Physical | Where-Object { ($_.MacAddress -replace '-','') -eq %q }\r\n",
			varName, strings.ToUpper(macNorm)))
		b.WriteString(fmt.Sprintf("if ($%s) {\r\n", varName))
		b.WriteString(fmt.Sprintf("    $guid%d = (Get-NetAdapterAdvancedProperty -InterfaceDescription $%s.InterfaceDescription -ErrorAction SilentlyContinue | Select-Object -First 1).InstanceID\r\n",
			i, varName))
		b.WriteString(fmt.Sprintf("    if (-not $guid%d) { $guid%d = $%s.InterfaceGuid }\r\n", i, i, varName))
		b.WriteString(fmt.Sprintf("    $regPath = \"HKLM:\\SYSTEM\\CurrentControlSet\\Services\\Tcpip\\Parameters\\Interfaces\\$guid%d\"\r\n", i))

		// Collect all IPs and masks
		allIPs := make([]string, 0, len(cfg.IPs))
		allMasks := make([]string, 0, len(cfg.IPs))
		for _, sip := range cfg.IPs {
			allIPs = append(allIPs, sip.IP)
			mask := sip.Netmask
			if mask == "" {
				mask = "255.255.255.0"
			}
			allMasks = append(allMasks, mask)
		}
		ipList := "'" + strings.Join(allIPs, "','") + "'"
		maskList := "'" + strings.Join(allMasks, "','") + "'"

		b.WriteString(fmt.Sprintf("    Set-ItemProperty -Path $regPath -Name IPAddress -Value @(%s) -Type MultiString\r\n", ipList))
		b.WriteString(fmt.Sprintf("    Set-ItemProperty -Path $regPath -Name SubnetMask -Value @(%s) -Type MultiString\r\n", maskList))
		b.WriteString("    Set-ItemProperty -Path $regPath -Name EnableDHCP -Value 0 -Type DWord\r\n")

		allDNS := collectDNS(cfg.IPs)
		if len(allDNS) > 0 {
			b.WriteString(fmt.Sprintf("    Set-ItemProperty -Path $regPath -Name NameServer -Value '%s' -Type String\r\n",
				strings.Join(allDNS, ",")))
		}
		b.WriteString("}\r\n\r\n")
	}
	return b.String()
}

func collectDNS(ips []types.StaticIP) []string {
	seen := make(map[string]bool)
	var result []string
	for _, sip := range ips {
		for _, d := range sip.DNS {
			if !seen[d] {
				seen[d] = true
				result = append(result, d)
			}
		}
	}
	return result
}

func netmaskToPrefix(mask string) string {
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
