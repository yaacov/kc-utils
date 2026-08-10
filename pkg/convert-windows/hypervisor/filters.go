package hypervisor

import (
	"log/slog"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/registry"
)

// RemoveFilter removes filterName from a REG_MULTI_SZ class filter list
// (UpperFilters / LowerFilters) when present.
func RemoveFilter(hive registry.Hive, path, valueName, filterName string) {
	vals, err := hive.GetMultiString(path, valueName)
	if err != nil {
		return
	}
	var kept []string
	changed := false
	for _, v := range vals {
		if v == "" {
			continue
		}
		if strings.EqualFold(v, filterName) {
			changed = true
			continue
		}
		kept = append(kept, v)
	}
	if !changed {
		return
	}
	hive.SetMultiString(path, valueName, kept)
	slog.Info("removed class filter", "path", path, "value", valueName, "filter", filterName)
}

// DisableService sets Start=4 when the service key exists.
func DisableService(hive registry.Hive, ccs, service string) {
	svcPath := ccs + `\Services\` + service
	if hive.KeyExists(svcPath) {
		hive.SetDWORD(svcPath, "Start", 4)
		slog.Info("disabled service", "service", service)
	}
}
