package pnputil

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"
)

type Plugin struct{}

func init() {
	firstboot.Contributors.Register("pnputil", &Plugin{})
}

func (p *Plugin) Priority() int { return 2000 }
func (p *Plugin) Name() string  { return "install-virtio-drivers" }

func (p *Plugin) ShouldRun(cfg *firstboot.ContributorConfig) bool {
	return len(cfg.DriverFiles) > 0
}

func (p *Plugin) Generate(cfg *firstboot.ContributorConfig) (string, error) {
	var lines []string
	lines = append(lines, "# Install VirtIO drivers via PnPutil")
	for _, df := range cfg.DriverFiles {
		infName := filepath.Base(df.InfPath)
		if !strings.HasSuffix(strings.ToLower(infName), ".inf") {
			continue
		}
		lines = append(lines,
			fmt.Sprintf(`pnputil /add-driver "C:\Windows\Drivers\VirtIO\%s" /install`, infName))
	}
	if len(lines) <= 1 {
		return "", nil
	}
	return strings.Join(lines, "\r\n") + "\r\n", nil
}
