package version

import (
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

type baseHandler struct {
	name string
}

func (b baseHandler) Name() string { return b.name }

type unknownHandler struct{ baseHandler }

func (unknownHandler) Matches(*types.InspectData) bool { return true }
func (unknownHandler) DriverOSPreferences() []string   { return nil }
func (unknownHandler) DriverOSFallbacks() []string     { return nil }
func (unknownHandler) FirstbootLauncher() LauncherKind { return LauncherModern }
func (unknownHandler) SupportsPowerShell() bool        { return true }
func (unknownHandler) StaticIPMode() StaticIPKind      { return StaticIPNetCmdlet }
func (unknownHandler) DiskOnlineMode() DiskOnlineKind  { return DiskOnlineGetDisk }
func (unknownHandler) VMwareCleanupMode() VMwareCleanupKind {
	return VMwareCleanupPNP
}
func (unknownHandler) SupportsQEMUGA() bool { return true }

var handlerUnknown VersionHandler = unknownHandler{baseHandler{name: "winunknown"}}

type win11 struct{ baseHandler }

func (win11) Matches(i *types.InspectData) bool {
	return i.MajorVersion >= 10 && isWindows11Product(i.ProductName)
}
func (win11) DriverOSPreferences() []string { return []string{"2k22", "w11"} }
func (win11) DriverOSFallbacks() []string   { return []string{"w10", "2k19"} }
func (win11) FirstbootLauncher() LauncherKind {
	return LauncherModern
}
func (win11) SupportsPowerShell() bool             { return true }
func (win11) StaticIPMode() StaticIPKind           { return StaticIPNetCmdlet }
func (win11) DiskOnlineMode() DiskOnlineKind       { return DiskOnlineGetDisk }
func (win11) VMwareCleanupMode() VMwareCleanupKind { return VMwareCleanupPNP }
func (win11) SupportsQEMUGA() bool                 { return true }

type win10 struct{ baseHandler }

func (win10) Matches(i *types.InspectData) bool {
	return i.MajorVersion >= 10 && !isWindows11Product(i.ProductName)
}
func (win10) DriverOSPreferences() []string { return []string{"w10", "2k19", "2k16"} }
func (win10) DriverOSFallbacks() []string   { return []string{"2k22"} }
func (win10) FirstbootLauncher() LauncherKind {
	return LauncherModern
}
func (win10) SupportsPowerShell() bool             { return true }
func (win10) StaticIPMode() StaticIPKind           { return StaticIPNetCmdlet }
func (win10) DiskOnlineMode() DiskOnlineKind       { return DiskOnlineGetDisk }
func (win10) VMwareCleanupMode() VMwareCleanupKind { return VMwareCleanupPNP }
func (win10) SupportsQEMUGA() bool                 { return true }

type win81 struct{ baseHandler }

func (win81) Matches(i *types.InspectData) bool {
	return i.MajorVersion == 6 && i.MinorVersion == 3
}
func (win81) DriverOSPreferences() []string { return []string{"w8.1", "2k12r2"} }
func (win81) DriverOSFallbacks() []string   { return nil }
func (win81) FirstbootLauncher() LauncherKind {
	return LauncherModern
}
func (win81) SupportsPowerShell() bool             { return true }
func (win81) StaticIPMode() StaticIPKind           { return StaticIPNetCmdlet }
func (win81) DiskOnlineMode() DiskOnlineKind       { return DiskOnlineGetDisk }
func (win81) VMwareCleanupMode() VMwareCleanupKind { return VMwareCleanupPNP }
func (win81) SupportsQEMUGA() bool                 { return true }

type win8 struct{ baseHandler }

func (win8) Matches(i *types.InspectData) bool {
	return i.MajorVersion == 6 && i.MinorVersion == 2
}
func (win8) DriverOSPreferences() []string { return []string{"w8", "2k12"} }
func (win8) DriverOSFallbacks() []string   { return nil }
func (win8) FirstbootLauncher() LauncherKind {
	return LauncherModern
}
func (win8) SupportsPowerShell() bool             { return true }
func (win8) StaticIPMode() StaticIPKind           { return StaticIPNetCmdlet }
func (win8) DiskOnlineMode() DiskOnlineKind       { return DiskOnlineGetDisk }
func (win8) VMwareCleanupMode() VMwareCleanupKind { return VMwareCleanupPNP }
func (win8) SupportsQEMUGA() bool                 { return true }

type win7 struct{ baseHandler }

func (win7) Matches(i *types.InspectData) bool {
	if i.MajorVersion == 6 && i.MinorVersion == 1 && !isServerProduct(i.ProductName) {
		return true
	}
	n := NormalizeProductName(i.ProductName)
	return i.MajorVersion == 0 && n != "" && !isServerProduct(n) && strings.Contains(n, "windows 7")
}
func (win7) DriverOSPreferences() []string { return []string{"w7", "2k8r2"} }
func (win7) DriverOSFallbacks() []string   { return nil }
func (win7) FirstbootLauncher() LauncherKind {
	return LauncherPSV1
}
func (win7) SupportsPowerShell() bool             { return true }
func (win7) StaticIPMode() StaticIPKind           { return StaticIPRegistry }
func (win7) DiskOnlineMode() DiskOnlineKind       { return DiskOnlineWMIDiskpart }
func (win7) VMwareCleanupMode() VMwareCleanupKind { return VMwareCleanupDevconBat }
func (win7) SupportsQEMUGA() bool                 { return true }

type win2008r2 struct{ baseHandler }

func (win2008r2) Matches(i *types.InspectData) bool {
	if i.MajorVersion == 6 && i.MinorVersion == 1 && isServerProduct(i.ProductName) {
		return true
	}
	n := NormalizeProductName(i.ProductName)
	return strings.Contains(n, "server 2008 r2") || strings.Contains(n, "2008 r2")
}
func (win2008r2) DriverOSPreferences() []string { return []string{"2k8r2", "w7"} }
func (win2008r2) DriverOSFallbacks() []string   { return nil }
func (win2008r2) FirstbootLauncher() LauncherKind {
	return LauncherPSV1
}
func (win2008r2) SupportsPowerShell() bool             { return true }
func (win2008r2) StaticIPMode() StaticIPKind           { return StaticIPRegistry }
func (win2008r2) DiskOnlineMode() DiskOnlineKind       { return DiskOnlineWMIDiskpart }
func (win2008r2) VMwareCleanupMode() VMwareCleanupKind { return VMwareCleanupDevconBat }
func (win2008r2) SupportsQEMUGA() bool                 { return true }

type win2008 struct{ baseHandler }

func (win2008) Matches(i *types.InspectData) bool {
	if i.MajorVersion == 6 && i.MinorVersion == 0 && isServerProduct(i.ProductName) {
		return true
	}
	n := NormalizeProductName(i.ProductName)
	return strings.Contains(n, "server 2008") && !strings.Contains(n, "2008 r2")
}
func (win2008) DriverOSPreferences() []string { return []string{"2k8", "vista"} }
func (win2008) DriverOSFallbacks() []string   { return nil }
func (win2008) FirstbootLauncher() LauncherKind {
	return LauncherPSV1
}
func (win2008) SupportsPowerShell() bool             { return true }
func (win2008) StaticIPMode() StaticIPKind           { return StaticIPWMINetsh }
func (win2008) DiskOnlineMode() DiskOnlineKind       { return DiskOnlineSkip }
func (win2008) VMwareCleanupMode() VMwareCleanupKind { return VMwareCleanupDevconBat }
func (win2008) SupportsQEMUGA() bool                 { return false }

type winvista struct{ baseHandler }

func (winvista) Matches(i *types.InspectData) bool {
	if i.MajorVersion == 6 && i.MinorVersion == 0 && !isServerProduct(i.ProductName) {
		return true
	}
	n := NormalizeProductName(i.ProductName)
	return n != "" && strings.Contains(n, "vista") && !isServerProduct(n)
}
func (winvista) DriverOSPreferences() []string { return []string{"vista", "2k8"} }
func (winvista) DriverOSFallbacks() []string   { return nil }
func (winvista) FirstbootLauncher() LauncherKind {
	return LauncherPSV1
}
func (winvista) SupportsPowerShell() bool             { return true }
func (winvista) StaticIPMode() StaticIPKind           { return StaticIPWMINetsh }
func (winvista) DiskOnlineMode() DiskOnlineKind       { return DiskOnlineWMIDiskpart }
func (winvista) VMwareCleanupMode() VMwareCleanupKind { return VMwareCleanupDevconBat }
func (winvista) SupportsQEMUGA() bool                 { return false }

type win2003 struct{ baseHandler }

func (win2003) Matches(i *types.InspectData) bool {
	if i.MajorVersion == 5 && i.MinorVersion == 2 {
		return true
	}
	n := NormalizeProductName(i.ProductName)
	return strings.Contains(n, "server 2003") || strings.Contains(n, "2003 server")
}
func (win2003) DriverOSPreferences() []string { return []string{"2k3", "xp"} }
func (win2003) DriverOSFallbacks() []string   { return []string{"xp"} }
func (win2003) FirstbootLauncher() LauncherKind {
	return LauncherBatOnly
}
func (win2003) SupportsPowerShell() bool             { return false }
func (win2003) StaticIPMode() StaticIPKind           { return StaticIPRegistry }
func (win2003) DiskOnlineMode() DiskOnlineKind       { return DiskOnlineWMIDiskpart }
func (win2003) VMwareCleanupMode() VMwareCleanupKind { return VMwareCleanupDevconBat }
func (win2003) SupportsQEMUGA() bool                 { return false }

type winxp struct{ baseHandler }

func (winxp) Matches(i *types.InspectData) bool {
	if i.MajorVersion == 5 && i.MinorVersion == 1 {
		return true
	}
	n := NormalizeProductName(i.ProductName)
	return strings.Contains(n, "windows xp") || n == "xp"
}
func (winxp) DriverOSPreferences() []string { return []string{"xp"} }
func (winxp) DriverOSFallbacks() []string   { return nil }
func (winxp) FirstbootLauncher() LauncherKind {
	return LauncherBatOnly
}
func (winxp) SupportsPowerShell() bool             { return false }
func (winxp) StaticIPMode() StaticIPKind           { return StaticIPRegistry }
func (winxp) DiskOnlineMode() DiskOnlineKind       { return DiskOnlineSkip }
func (winxp) VMwareCleanupMode() VMwareCleanupKind { return VMwareCleanupDevconBat }
func (winxp) SupportsQEMUGA() bool                 { return false }
