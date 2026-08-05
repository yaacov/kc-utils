package crashcontrol

import "github.com/yaacov/kc-utils/pkg/common/registry"

// Disable sets AutoReboot to 0 in the crash control registry key.
func Disable(systemHive registry.Hive, ccs string) {
	crashPath := ccs + `\Control\CrashControl`
	systemHive.CreateKey(crashPath)
	systemHive.SetDWORD(crashPath, "AutoReboot", 0)
}
