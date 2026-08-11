package drivers

// SCSIClassGUID is the Windows SCSI adapter class GUID.
const SCSIClassGUID = "{4D36E97B-E325-11CE-BFC1-08002BE10318}"

// BootCriticalDrivers lists storage drivers that must start at boot (Start=0)
// and be copied into system32\drivers for early load.
var BootCriticalDrivers = map[string]bool{
	"viostor": true,
	"vioscsi": true,
}

// PCIIDPair holds legacy and modern VirtIO PCI IDs for a storage driver.
type PCIIDPair struct {
	Legacy string
	Modern string
}

// StoragePCIIDs maps boot-critical storage drivers to virt-v2v PCI IDs
// (without the PCI# prefix used in CriticalDeviceDatabase keys).
var StoragePCIIDs = map[string]PCIIDPair{
	"viostor": {
		Legacy: "VEN_1AF4&DEV_1001&REV_00",
		Modern: "VEN_1AF4&DEV_1042&REV_01",
	},
	"vioscsi": {
		Legacy: "VEN_1AF4&DEV_1004&REV_00",
		Modern: "VEN_1AF4&DEV_1048&REV_01",
	},
}
