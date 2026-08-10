package qemuga

import (
	"path/filepath"
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
	return qemuGAMSIName(cfg) != ""
}

func (p *Plugin) UsesBatch(_ *firstboot.ContributorConfig) bool { return false }

func (p *Plugin) Generate(cfg *firstboot.ContributorConfig) (string, error) {
	msiName := qemuGAMSIName(cfg)
	if msiName == "" {
		return "", nil
	}
	msiPath := `C:\Windows\Drivers\VirtIO\` + msiName
	return "# Install QEMU Guest Agent\r\n" +
		"$msiPath = '" + msiPath + "'\r\n" +
		"if (Test-Path $msiPath) {\r\n" +
		"    Start-Process msiexec.exe -ArgumentList \"/i\", $msiPath, \"/qn\", \"/norestart\" -Wait\r\n" +
		"}\r\n", nil
}

func qemuGAMSIName(cfg *firstboot.ContributorConfig) string {
	if cfg == nil {
		return ""
	}
	for _, df := range cfg.DriverFiles {
		if df.Name != "qemu-ga" && !strings.Contains(strings.ToLower(df.InfPath), "qemu-ga") {
			continue
		}
		name := filepath.Base(df.InfPath)
		if strings.HasSuffix(strings.ToLower(name), ".msi") {
			return name
		}
	}
	return ""
}
