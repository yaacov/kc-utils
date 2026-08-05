package criticaldb

import (
	"fmt"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/convert-windows/drivers"
)

type CriticalDBRegistrar struct{}

func init() {
	drivers.Registrars.Register("criticaldb", &CriticalDBRegistrar{})
}

func (c *CriticalDBRegistrar) Register(hive registry.Hive, ccs string, driverName, driverPath, group, arch string) error {
	svcPath := fmt.Sprintf("%s\\Services\\%s", ccs, driverName)
	hive.CreateKey(svcPath)
	hive.SetDWORD(svcPath, "Type", 1)
	hive.SetDWORD(svcPath, "Start", 0)
	hive.SetDWORD(svcPath, "ErrorControl", 1)
	hive.SetString(svcPath, "Group", group)
	hive.SetExpandString(svcPath, "ImagePath", driverPath)

	pair, ok := drivers.StoragePCIIDs[driverName]
	if !ok {
		return nil
	}
	for _, pciid := range []string{pair.Legacy, pair.Modern} {
		// virt-v2v uses a single key name "PCI#VEN_…&REV_…" under CriticalDeviceDatabase.
		cdbPath := fmt.Sprintf("%s\\Control\\CriticalDeviceDatabase\\PCI#%s", ccs, pciid)
		hive.CreateKey(cdbPath)
		hive.SetString(cdbPath, "ClassGUID", drivers.SCSIClassGUID)
		hive.SetString(cdbPath, "Service", driverName)
	}
	return nil
}
