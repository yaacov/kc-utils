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

func (p *Plugin) Generate(_ *firstboot.ContributorConfig) (string, error) {
	return staticip.RebootSignalScript(), nil
}

func (p *Plugin) UsesBatch(_ *firstboot.ContributorConfig) bool { return false }
