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

func (p *Plugin) UsesBatch(cfg *firstboot.ContributorConfig) bool {
	return cfg.Version != nil && !cfg.Version.SupportsPowerShell()
}

func (p *Plugin) Generate(cfg *firstboot.ContributorConfig) (string, error) {
	mode := version.DiskOnlineGetDisk
	if cfg.Version != nil {
		mode = cfg.Version.DiskOnlineMode()
	}
	if p.UsesBatch(cfg) {
		return diskpartBatScript(), nil
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

// diskpartBatScript brings fixed disks online on pre-PowerShell guests (e.g. Win2003).
// diskpart on those releases accepts an interactive script via stdin redirection.
// Read-only is cleared per volume with ATTRIBUTES VOLUME (Win2003-compatible);
// ATTRIBUTES DISK is not used on these releases.
func diskpartBatScript() string {
	return "@echo off\r\n" +
		"REM Bring offline disks online via diskpart\r\n" +
		"for /L %%i in (0,1,15) do (\r\n" +
		"  >\"%TEMP%\\kc-disk-%%i.txt\" echo select disk %%i\r\n" +
		"  >>\"%TEMP%\\kc-disk-%%i.txt\" echo online disk\r\n" +
		"  >>\"%TEMP%\\kc-disk-%%i.txt\" echo exit\r\n" +
		"  diskpart /s \"%TEMP%\\kc-disk-%%i.txt\" >nul 2>&1\r\n" +
		"  del \"%TEMP%\\kc-disk-%%i.txt\" >nul 2>&1\r\n" +
		")\r\n" +
		"REM Clear volume read-only flags (ATTRIBUTES VOLUME is Win2003-compatible)\r\n" +
		"for /L %%v in (0,1,15) do (\r\n" +
		"  >\"%TEMP%\\kc-vol-%%v.txt\" echo select volume %%v\r\n" +
		"  >>\"%TEMP%\\kc-vol-%%v.txt\" echo attributes volume clear readonly\r\n" +
		"  >>\"%TEMP%\\kc-vol-%%v.txt\" echo exit\r\n" +
		"  diskpart /s \"%TEMP%\\kc-vol-%%v.txt\" >nul 2>&1\r\n" +
		"  del \"%TEMP%\\kc-vol-%%v.txt\" >nul 2>&1\r\n" +
		")\r\n"
}
