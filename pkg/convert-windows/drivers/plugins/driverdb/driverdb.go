package driverdb

import (
	"fmt"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/convert-windows/drivers"
	"github.com/yaacov/kc-utils/pkg/convert-windows/driversource"
)

type DriverDBRegistrar struct{}

func init() {
	drivers.Registrars.Register("driverdb", &DriverDBRegistrar{})
}

func (d *DriverDBRegistrar) Register(hive registry.Hive, ccs string, driverName, driverPath, group, arch string) error {
	svcPath := fmt.Sprintf("%s\\Services\\%s", ccs, driverName)
	start := uint32(3)
	if drivers.BootCriticalDrivers[driverName] {
		start = 0
	}
	if !hive.KeyExists(svcPath) {
		hive.CreateKey(svcPath)
		hive.SetDWORD(svcPath, "Type", 1)
		hive.SetDWORD(svcPath, "Start", start)
		hive.SetDWORD(svcPath, "ErrorControl", 1)
		hive.SetString(svcPath, "Group", group)
		hive.SetExpandString(svcPath, "ImagePath", driverPath)
	} else if drivers.BootCriticalDrivers[driverName] {
		hive.SetDWORD(svcPath, "Start", 0)
	}

	pair, ok := drivers.StoragePCIIDs[driverName]
	if !ok {
		return nil
	}

	winarch := ddbArch(arch)
	// Per-driver INF/package names so registering viostor and vioscsi does not
	// overwrite each other's DriverDatabase Service binding.
	drvInf := strings.ToLower(driverName) + ".inf"
	drvInfLabel := fmt.Sprintf("%s_%s_0000000000000000", drvInf, winarch)
	drvConfig := strings.ToLower(driverName) + "_conf"

	infPath := `DriverDatabase\DriverInfFiles\` + drvInf
	hive.CreateKey(infPath)
	hive.SetMultiString(infPath, "", []string{drvInfLabel})
	hive.SetString(infPath, "Active", drvInfLabel)
	hive.SetMultiString(infPath, "Configurations", []string{drvConfig})

	for _, pciid := range []string{pair.Legacy, pair.Modern} {
		devPath := `DriverDatabase\DeviceIds\PCI\` + pciid
		hive.CreateKey(devPath)
		hive.SetBinary(devPath, drvInf, []byte{0x01, 0xff, 0x00, 0x00})

		pkgPath := `DriverDatabase\DriverPackages\` + drvInfLabel
		hive.CreateKey(pkgPath)
		hive.SetBinary(pkgPath, "Version", ddbVersionBlob())

		cfgPath := pkgPath + `\Configurations\` + drvConfig
		hive.CreateKey(cfgPath)
		hive.SetDWORD(cfgPath, "ConfigFlags", 0)
		hive.SetString(cfgPath, "Service", driverName)

		descPath := pkgPath + `\Descriptors\PCI\` + pciid
		hive.CreateKey(descPath)
		hive.SetString(descPath, "Configuration", drvConfig)
	}
	return nil
}

func ddbArch(arch string) string {
	switch driversource.NormalizeArch(arch) {
	case "x86":
		return "x86"
	default:
		return "amd64"
	}
}

// ddbVersionBlob matches virt-v2v's DriverPackages Version REG_BINARY for viostor.
func ddbVersionBlob() []byte {
	// \x00\xff\x09\x00\x00\x00\x00\x00 + SCSI class GUID bytes + 24 zero bytes
	guid := []byte{
		0x7b, 0xe9, 0x36, 0x4d, 0x25, 0xe3, 0xce, 0x11,
		0xbf, 0xc1, 0x08, 0x00, 0x2b, 0xe1, 0x03, 0x18,
	}
	blob := make([]byte, 0, 8+len(guid)+24)
	blob = append(blob, 0x00, 0xff, 0x09, 0x00, 0x00, 0x00, 0x00, 0x00)
	blob = append(blob, guid...)
	blob = append(blob, make([]byte, 24)...)
	return blob
}
