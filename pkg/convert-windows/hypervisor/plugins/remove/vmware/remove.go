//go:build linux

package vmware

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"
	"github.com/yaacov/kc-utils/pkg/convert-windows/hypervisor"
	"github.com/yaacov/kc-utils/pkg/guest"
)

const (
	uninstallKey      = `Microsoft\Windows\CurrentVersion\Uninstall\VMware Tools`
	installerProducts = `Classes\Installer\Products`
	installerFeatures = `Classes\Installer\Features`
	userDataProducts  = `Microsoft\Windows\CurrentVersion\Installer\UserData\S-1-5-18\Products`
	taskCacheTree     = `Microsoft\Windows NT\CurrentVersion\Schedule\TaskCache\Tree\VMware`
)

type Remove struct{}

func init() {
	hypervisor.WindowsRemoves.Register("vmware", &Remove{})
}

func (r *Remove) Detect(guestRoot string, _, softwareHive registry.Hive) bool {
	toolsDir := filepath.Join(guestRoot, "Program Files", "VMware", "VMware Tools")
	return guest.FileExists(toolsDir) || softwareHive.KeyExists(uninstallKey)
}

func (r *Remove) Remove(guestRoot string, _, softwareHive registry.Hive) error {
	toolsDir := filepath.Join(guestRoot, "Program Files", "VMware", "VMware Tools")
	_ = guest.FileRemoveAll(toolsDir)
	softwareHive.DeleteKey(uninstallKey)

	guids := removeMSIProducts(softwareHive)
	removeScheduledTasks(softwareHive)
	writeMSIUninstallFirstboot(guestRoot, guids)

	return nil
}

// removeMSIProducts removes Windows Installer product/feature registry entries
// for VMware Tools and returns the encoded GUIDs that were found.
func removeMSIProducts(hive registry.Hive) []string {
	var guids []string

	keys, err := hive.EnumKeys(installerProducts)
	if err == nil {
		for _, guid := range keys {
			name, _ := hive.GetString(installerProducts+`\`+guid, "ProductName")
			if !strings.Contains(strings.ToLower(name), "vmware tools") {
				continue
			}
			slog.Info("removing MSI product entry", "guid", guid, "name", name)
			guids = append(guids, guid)
			hive.DeleteKey(installerProducts + `\` + guid)
			if hive.KeyExists(installerFeatures + `\` + guid) {
				hive.DeleteKey(installerFeatures + `\` + guid)
			}
		}
	}

	udKeys, err := hive.EnumKeys(userDataProducts)
	if err == nil {
		for _, guid := range udKeys {
			instProps := userDataProducts + `\` + guid + `\InstallProperties`
			name, _ := hive.GetString(instProps, "DisplayName")
			if !strings.Contains(strings.ToLower(name), "vmware tools") {
				continue
			}
			slog.Info("removing MSI user-data entry", "guid", guid)
			hive.DeleteKey(userDataProducts + `\` + guid)
		}
	}

	return guids
}

// removeScheduledTasks removes VMware scheduled task entries from the registry.
func removeScheduledTasks(hive registry.Hive) {
	if hive.KeyExists(taskCacheTree) {
		slog.Info("removing VMware scheduled task entries")
		hive.DeleteKey(taskCacheTree)
	}
}

// writeMSIUninstallFirstboot writes a firstboot PowerShell script that
// silently uninstalls any remaining VMware Tools MSI products.
func writeMSIUninstallFirstboot(guestRoot string, guids []string) {
	var script strings.Builder
	script.WriteString("# Silent uninstall of VMware Tools MSI remnants\r\n")
	script.WriteString("$ErrorActionPreference = 'SilentlyContinue'\r\n")

	if len(guids) > 0 {
		for _, encoded := range guids {
			decoded := decodeMSIGUID(encoded)
			if decoded == "" {
				continue
			}
			script.WriteString(fmt.Sprintf(
				"Start-Process msiexec.exe -ArgumentList '/x %s /qn /norestart' -Wait -NoNewWindow\r\n",
				decoded,
			))
		}
	}

	script.WriteString("Get-WmiObject Win32_Product | Where-Object { $_.Name -like '*VMware Tools*' } | ForEach-Object { $_.Uninstall() }\r\n")

	script.WriteString("\r\n# Stop and delete residual VMware services\r\n")
	script.WriteString("Get-Service | Where-Object { $_.Name -like '*VMware*' -or $_.Name -like '*VMTools*' -or $_.Name -like '*VGAuth*' } | Stop-Service -Force -ErrorAction SilentlyContinue\r\n")
	script.WriteString("Get-Service | Where-Object { $_.Name -like '*VMware*' -or $_.Name -like '*VMTools*' -or $_.Name -like '*VGAuth*' } | ForEach-Object { sc.exe delete $_.Name } 2>&1 | Out-Null\r\n")

	firstbootDir := filepath.Join(guestRoot, "Program Files", "Guestfs", "Firstboot")
	if err := firstboot.WriteScript(firstbootDir, 10, "vmware-msi-uninstall", script.String()); err != nil {
		slog.Warn("failed to write VMware MSI uninstall firstboot script", "error", err)
	}
}

// decodeMSIGUID converts a Windows Installer encoded GUID back to standard
// form: {XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}.
// Encoded format reverses each group: "87654321432143214321CBA987654321"
// becomes "{12345678-1234-1234-1234-123456789ABC}".
func decodeMSIGUID(encoded string) string {
	if len(encoded) != 32 {
		return ""
	}
	reverse := func(s string) string {
		r := []byte(s)
		for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
			r[i], r[j] = r[j], r[i]
		}
		return string(r)
	}
	pairReverse := func(s string) string {
		r := make([]byte, len(s))
		for i := 0; i+1 < len(s); i += 2 {
			r[i] = s[i+1]
			r[i+1] = s[i]
		}
		return string(r)
	}

	return fmt.Sprintf("{%s-%s-%s-%s-%s}",
		reverse(encoded[0:8]),
		reverse(encoded[8:12]),
		reverse(encoded[12:16]),
		pairReverse(encoded[16:20]),
		pairReverse(encoded[20:32]),
	)
}
