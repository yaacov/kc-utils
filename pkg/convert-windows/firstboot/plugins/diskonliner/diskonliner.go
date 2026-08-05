package diskonliner

import (
	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"
	"github.com/yaacov/kc-utils/pkg/convert-windows/version"
)

type Plugin struct{}

func init() {
	firstboot.Contributors.Register("diskonliner", &Plugin{})
}

func (p *Plugin) Priority() int { return 4000 }
func (p *Plugin) Name() string  { return "disk-onliner" }

func (p *Plugin) ShouldRun(cfg *firstboot.ContributorConfig) bool {
	if cfg.Version == nil {
		return true
	}
	return cfg.Version.DiskOnlineMode() != version.DiskOnlineSkip
}

func (p *Plugin) UsesBatch(_ *firstboot.ContributorConfig) bool { return false }

func (p *Plugin) Generate(cfg *firstboot.ContributorConfig) (string, error) {
	mode := version.DiskOnlineGetDisk
	if cfg.Version != nil {
		mode = cfg.Version.DiskOnlineMode()
	}
	switch mode {
	case version.DiskOnlineWMIDiskpart:
		return wmiDiskpartScript(), nil
	default:
		return "# Bring all offline disks online\r\n" +
			"Get-Disk | Where-Object IsOffline | Set-Disk -IsOffline $false\r\n" +
			"Get-Disk | Where-Object IsReadOnly | Set-Disk -IsReadOnly $false\r\n", nil
	}
}

func wmiDiskpartScript() string {
	return "# Bring offline disks online via diskpart\r\n" +
		"$script = @'\r\n" +
		"select disk %d\r\n" +
		"online disk\r\n" +
		"attributes disk clear readonly\r\n" +
		"'@\r\n" +
		"Get-WmiObject -Query \"SELECT * FROM Win32_DiskDrive WHERE MediaType='Fixed hard disk media'\" | ForEach-Object {\r\n" +
		"    $idx = $_.Index\r\n" +
		"    $cmd = $script -f $idx\r\n" +
		"    $cmd | diskpart.exe\r\n" +
		"}\r\n"
}
