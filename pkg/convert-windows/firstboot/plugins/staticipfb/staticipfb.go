package staticipfb

import (
	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"
	"github.com/yaacov/kc-utils/pkg/convert-windows/staticip"
)

type Plugin struct{}

func init() {
	firstboot.Contributors.Register("staticip", &Plugin{})
}

func (p *Plugin) Priority() int { return 2500 }
func (p *Plugin) Name() string  { return "static-ip" }

func (p *Plugin) ShouldRun(cfg *firstboot.ContributorConfig) bool {
	return len(cfg.StaticIPs) > 0
}

func (p *Plugin) Generate(cfg *firstboot.ContributorConfig) (string, error) {
	var content string
	if cfg.Options.WindowsRegistryNetwork {
		content = staticip.RegistryScript(cfg.StaticIPs)
	} else {
		content = staticip.PowerShellScript(cfg.StaticIPs)
	}
	if content == "" {
		return "", nil
	}
	waitAndConfigure := "# Wait for virtio-net then configure static IPs\r\n" +
		"$deadline = (Get-Date).AddMinutes(5)\r\n" +
		"while ((Get-Date) -lt $deadline) {\r\n" +
		"    if (Get-NetAdapter | Where-Object { $_.DriverFileName -like '*netkvm*' -or $_.InterfaceDescription -like '*VirtIO*' }) { break }\r\n" +
		"    Start-Sleep -Seconds 5\r\n" +
		"}\r\n" + content
	return waitAndConfigure, nil
}
