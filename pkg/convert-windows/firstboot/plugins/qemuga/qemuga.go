package qemuga

import (
	"strings"

	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"
)

type Plugin struct{}

func init() {
	firstboot.Contributors.Register("qemuga", &Plugin{})
}

func (p *Plugin) Priority() int { return 3000 }
func (p *Plugin) Name() string  { return "install-qemu-ga" }

func (p *Plugin) ShouldRun(cfg *firstboot.ContributorConfig) bool {
	if cfg.Version != nil && !cfg.Version.SupportsQEMUGA() {
		return false
	}
	for _, df := range cfg.DriverFiles {
		if df.Name == "qemu-ga" || strings.Contains(strings.ToLower(df.InfPath), "qemu-ga") {
			return true
		}
	}
	return false
}

func (p *Plugin) UsesBatch(_ *firstboot.ContributorConfig) bool { return false }

func (p *Plugin) Generate(_ *firstboot.ContributorConfig) (string, error) {
	return "# Install QEMU Guest Agent\r\n" +
		"$msiPath = (Get-ChildItem \"C:\\Windows\\Drivers\\VirtIO\\qemu-ga*.msi\"" +
		" -ErrorAction SilentlyContinue | Select-Object -First 1).FullName\r\n" +
		"if ($msiPath) {\r\n" +
		"    Start-Process msiexec.exe -ArgumentList \"/i\", $msiPath, \"/qn\", \"/norestart\" -Wait\r\n" +
		"}\r\n", nil
}
