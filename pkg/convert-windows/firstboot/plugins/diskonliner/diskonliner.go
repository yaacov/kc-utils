package diskonliner

import "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"

type Plugin struct{}

func init() {
	firstboot.Contributors.Register("diskonliner", &Plugin{})
}

func (p *Plugin) Priority() int { return 4000 }
func (p *Plugin) Name() string  { return "disk-onliner" }

func (p *Plugin) ShouldRun(_ *firstboot.ContributorConfig) bool { return true }

func (p *Plugin) Generate(_ *firstboot.ContributorConfig) (string, error) {
	return "# Bring all offline disks online\r\n" +
		"Get-Disk | Where-Object IsOffline | Set-Disk -IsOffline $false\r\n" +
		"Get-Disk | Where-Object IsReadOnly | Set-Disk -IsReadOnly $false\r\n", nil
}
