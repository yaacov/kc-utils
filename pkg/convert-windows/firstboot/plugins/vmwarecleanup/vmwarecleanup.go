package vmwarecleanup

import (
	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"
	"github.com/yaacov/kc-utils/pkg/convert-windows/staticip"
)

type Plugin struct{}

func init() {
	firstboot.Contributors.Register("vmwarecleanup", &Plugin{})
}

func (p *Plugin) Priority() int { return 9100 }
func (p *Plugin) Name() string  { return "cleanup-vmware" }

func (p *Plugin) ShouldRun(cfg *firstboot.ContributorConfig) bool {
	return cfg.Options.VMwareDriverRemoval
}

func (p *Plugin) Generate(_ *firstboot.ContributorConfig) (string, error) {
	return staticip.VMwareCleanupScript(), nil
}
