package qemuga

import "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"

type Plugin struct{}

func init() {
	firstboot.Contributors.Register("qemuga", &Plugin{})
}

func (p *Plugin) Priority() int { return 3000 }
func (p *Plugin) Name() string  { return "install-qemu-ga" }

func (p *Plugin) ShouldRun(_ *firstboot.ContributorConfig) bool { return true }

func (p *Plugin) Generate(_ *firstboot.ContributorConfig) (string, error) {
	return "# Install QEMU Guest Agent\r\n" +
		"$msiPath = (Get-ChildItem \"C:\\Windows\\Drivers\\VirtIO\\qemu-ga*.msi\"" +
		" -ErrorAction SilentlyContinue | Select-Object -First 1).FullName\r\n" +
		"if ($msiPath) {\r\n" +
		"    Start-Process msiexec.exe -ArgumentList \"/i\", $msiPath, \"/qn\", \"/norestart\" -Wait\r\n" +
		"}\r\n", nil
}
