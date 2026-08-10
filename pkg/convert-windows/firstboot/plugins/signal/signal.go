package signal

import (
	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"
	"github.com/yaacov/kc-utils/pkg/convert-windows/staticip"
)

type Plugin struct{}

func init() {
	firstboot.Contributors.Register("signal", &Plugin{})
}

func (p *Plugin) Priority() int { return 99999 }
func (p *Plugin) Name() string  { return "signal-conversion-done" }

func (p *Plugin) ShouldRun(cfg *firstboot.ContributorConfig) bool {
	return cfg.Options.WaitForGuestReboot
}

func (p *Plugin) UsesBatch(cfg *firstboot.ContributorConfig) bool {
	return cfg.Version != nil && !cfg.Version.SupportsPowerShell()
}

func (p *Plugin) Generate(cfg *firstboot.ContributorConfig) (string, error) {
	if p.UsesBatch(cfg) {
		return staticip.RebootSignalBatScript(), nil
	}
	return staticip.RebootSignalScript(), nil
}
