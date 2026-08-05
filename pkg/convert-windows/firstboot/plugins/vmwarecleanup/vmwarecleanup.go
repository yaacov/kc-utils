package vmwarecleanup

import (
	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"
	"github.com/yaacov/kc-utils/pkg/convert-windows/staticip"
	"github.com/yaacov/kc-utils/pkg/convert-windows/version"
)

type Plugin struct{}

func init() {
	firstboot.Contributors.Register("vmwarecleanup", &Plugin{})
}

func (p *Plugin) Priority() int { return 9100 }
func (p *Plugin) Name() string  { return "cleanup-vmware" }

func (p *Plugin) ShouldRun(cfg *firstboot.ContributorConfig) bool {
	if !cfg.Options.VMwareDriverRemoval {
		return false
	}
	if cfg.Version != nil && cfg.Version.VMwareCleanupMode() == version.VMwareCleanupSkip {
		return false
	}
	return true
}

func (p *Plugin) UsesBatch(cfg *firstboot.ContributorConfig) bool {
	return cfg.Version != nil && cfg.Version.VMwareCleanupMode() == version.VMwareCleanupDevconBat
}

func (p *Plugin) Generate(cfg *firstboot.ContributorConfig) (string, error) {
	if p.UsesBatch(cfg) {
		return staticip.DevconVMwareCleanupBat(), nil
	}
	return staticip.VMwareCleanupScript(), nil
}
