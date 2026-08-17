//go:build linux

package firstboot_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	regmock "github.com/yaacov/kc-utils/pkg/common/registry/mock"
	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-windows/driversource"
	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"
	"github.com/yaacov/kc-utils/pkg/convert-windows/version"

	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/diskonliner"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/multipleips"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/pnputil"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/qemuga"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/routecleanup"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/signal"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/staticipfb"
	_ "github.com/yaacov/kc-utils/pkg/convert-windows/firstboot/plugins/vmwarecleanup"
)

// setupFakeSrvany creates a temp directory with a dummy rhsrvany.exe and sets
// KC_VIRT_TOOLS to point there. Returns a cleanup function.
func setupFakeSrvany(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "rhsrvany.exe"), []byte("FAKE_EXE"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KC_VIRT_TOOLS", dir)
}

func TestWriteScript(t *testing.T) {
	tmpDir := t.TempDir()
	content := "# test script\r\necho hello\r\n"
	err := firstboot.WriteScript(tmpDir, 2000, "test-script", content)
	if err != nil {
		t.Fatalf("WriteScript returned error: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "scripts", "2000-test-script.ps1")
	info, err := os.Stat(expectedPath)
	if err != nil {
		t.Fatalf("expected file %s to exist, got error: %v", expectedPath, err)
	}
	if info.Size() == 0 {
		t.Errorf("expected non-empty file, got size 0")
	}

	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("reading script file: %v", err)
	}
	if string(data) != content {
		t.Errorf("file content mismatch: got %q, want %q", string(data), content)
	}
}

func TestConfigureOfflineStillAddsQemuGAScript(t *testing.T) {
	setupFakeSrvany(t)
	root := t.TempDir()
	hive := regmock.NewMockHive()

	h := version.Classify(&types.InspectData{
		MajorVersion: 10,
		ProductName:  "Windows Server 2022",
	})

	err := firstboot.Configure(&firstboot.Config{
		MountRoot: root,
		Offline:   true,
		Version:   h,
		DriverFiles: []driversource.DriverFile{{
			Name:    "qemu-ga",
			InfPath: "/usr/share/virtio-win/guest-agent/qemu-ga-x86_64.msi",
		}},
	}, hive, "ControlSet001")
	if err != nil {
		t.Fatalf("Configure returned error: %v", err)
	}

	scriptPath := filepath.Join(root, "Program Files", "Guestfs", "Firstboot", "scripts", "3000-install-qemu-ga.ps1")
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("expected qemu-ga script to exist: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `C:\Windows\Drivers\VirtIO\qemu-ga-x86_64.msi`) {
		t.Fatalf("expected exact MSI path, got: %s", content)
	}
	if strings.Contains(content, "Get-ChildItem") {
		t.Fatalf("script should not glob MSI files: %s", content)
	}
}

func TestConfigureWin2008UsesPSV1Launcher(t *testing.T) {
	setupFakeSrvany(t)
	root := t.TempDir()
	hive := regmock.NewMockHive()

	h := version.Classify(&types.InspectData{
		MajorVersion: 6,
		MinorVersion: 0,
		ProductName:  "Windows Server (R) 2008 Enterprise",
	})

	err := firstboot.Configure(&firstboot.Config{
		MountRoot: root,
		Offline:   true,
		Version:   h,
		DriverFiles: []driversource.DriverFile{{
			Name:    "viostor",
			InfPath: "viostor.inf",
		}},
	}, hive, "ControlSet001")
	if err != nil {
		t.Fatalf("Configure returned error: %v", err)
	}

	bat, err := os.ReadFile(filepath.Join(root, "Program Files", "Guestfs", "Firstboot", "firstboot.bat"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(bat)
	if !strings.Contains(content, "ExecutionPolicy") {
		t.Fatalf("expected PS 1.0 launcher, got: %s", content)
	}
	if !strings.Contains(content, `C:\Windows\System32\shutdown.exe /r /t 5 /f`) {
		t.Fatalf("expected post-install reboot in launcher, got: %s", content)
	}

	diskScript := filepath.Join(root, "Program Files", "Guestfs", "Firstboot", "scripts", "4000-disk-onliner.ps1")
	if _, err := os.Stat(diskScript); err == nil {
		t.Fatal("win2008 should skip disk-onliner contributor")
	}
}

func TestConfigureModernLauncherRebootsAfterCleanup(t *testing.T) {
	setupFakeSrvany(t)
	root := t.TempDir()
	hive := regmock.NewMockHive()

	h := version.Classify(&types.InspectData{
		MajorVersion: 10,
		ProductName:  "Windows Server 2022",
	})

	err := firstboot.Configure(&firstboot.Config{
		MountRoot: root,
		Offline:   true,
		Version:   h,
		Options:   types.PrepareOptions{WaitForGuestReboot: true},
		DriverFiles: []driversource.DriverFile{{
			Name:    "viostor",
			InfPath: "viostor.inf",
		}},
	}, hive, "ControlSet001")
	if err != nil {
		t.Fatalf("Configure returned error: %v", err)
	}

	bat, err := os.ReadFile(filepath.Join(root, "Program Files", "Guestfs", "Firstboot", "firstboot.bat"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(bat)
	cleanupIdx := strings.Index(content, `rmdir /s /q "C:\Program Files\Guestfs\Firstboot"`)
	rebootIdx := strings.Index(content, `C:\Windows\System32\shutdown.exe /r /t 5 /f`)
	if cleanupIdx < 0 || rebootIdx < 0 {
		t.Fatalf("expected reboot then cleanup in launcher, got: %s", content)
	}
	if rebootIdx > cleanupIdx {
		t.Fatalf("reboot must be scheduled before deleting Firstboot: %s", content)
	}

	signalPath := filepath.Join(root, "Program Files", "Guestfs", "Firstboot", "scripts", "99999-signal-conversion-done.ps1")
	signal, err := os.ReadFile(signalPath)
	if err != nil {
		t.Fatalf("expected COM1 signal script before footer reboot: %v", err)
	}
	if !strings.Contains(string(signal), "CONVERSION_DONE") {
		t.Fatalf("signal script missing CONVERSION_DONE: %s", signal)
	}
}

func TestConfigureXPWritesBatchSignalScript(t *testing.T) {
	setupFakeSrvany(t)
	root := t.TempDir()
	hive := regmock.NewMockHive()

	h := version.Classify(&types.InspectData{
		MajorVersion: 5,
		MinorVersion: 1,
		ProductName:  "Windows XP Professional",
	})

	err := firstboot.Configure(&firstboot.Config{
		MountRoot: root,
		Offline:   true,
		Version:   h,
		Options:   types.PrepareOptions{WaitForGuestReboot: true},
	}, hive, "ControlSet001")
	if err != nil {
		t.Fatalf("Configure returned error: %v", err)
	}

	signalPath := filepath.Join(root, "Program Files", "Guestfs", "Firstboot", "scripts", "99999-signal-conversion-done.bat")
	signal, err := os.ReadFile(signalPath)
	if err != nil {
		t.Fatalf("expected batch COM1 signal script for XP: %v", err)
	}
	if !strings.Contains(string(signal), "CONVERSION_DONE") {
		t.Fatalf("signal script missing CONVERSION_DONE: %s", signal)
	}
	psPath := filepath.Join(root, "Program Files", "Guestfs", "Firstboot", "scripts", "99999-signal-conversion-done.ps1")
	if _, err := os.Stat(psPath); err == nil {
		t.Fatal("XP must not write PowerShell signal script")
	}
}

func TestConfigureRegistersWindowsService(t *testing.T) {
	setupFakeSrvany(t)
	root := t.TempDir()
	hive := regmock.NewMockHive()

	h := version.Classify(&types.InspectData{
		MajorVersion: 10,
		ProductName:  "Windows Server 2022",
	})

	err := firstboot.Configure(&firstboot.Config{
		MountRoot: root,
		Offline:   true,
		Version:   h,
		DriverFiles: []driversource.DriverFile{{
			Name:    "viostor",
			InfPath: "viostor.inf",
		}},
	}, hive, "ControlSet001")
	if err != nil {
		t.Fatalf("Configure returned error: %v", err)
	}

	svcPath := `ControlSet001\Services\kcfirstboot`
	if !hive.KeyExists(svcPath) {
		t.Fatal("expected kcfirstboot service key to exist")
	}

	typ, err := hive.GetDWORD(svcPath, "Type")
	if err != nil || typ != 0x10 {
		t.Fatalf("expected Type=0x10, got %d err=%v", typ, err)
	}
	start, err := hive.GetDWORD(svcPath, "Start")
	if err != nil || start != 0x02 {
		t.Fatalf("expected Start=0x02, got %d err=%v", start, err)
	}
	errCtl, err := hive.GetDWORD(svcPath, "ErrorControl")
	if err != nil || errCtl != 0x01 {
		t.Fatalf("expected ErrorControl=0x01, got %d err=%v", errCtl, err)
	}
	imgPath, err := hive.GetString(svcPath, "ImagePath")
	if err != nil {
		t.Fatalf("GetString ImagePath: %v", err)
	}
	if !strings.Contains(imgPath, "rhsrvany.exe") || !strings.Contains(imgPath, "kcfirstboot") {
		t.Fatalf("unexpected ImagePath: %s", imgPath)
	}
	objName, err := hive.GetString(svcPath, "ObjectName")
	if err != nil || objName != "LocalSystem" {
		t.Fatalf("expected ObjectName=LocalSystem, got %q err=%v", objName, err)
	}

	paramsPath := svcPath + `\Parameters`
	cmdLine, err := hive.GetString(paramsPath, "CommandLine")
	if err != nil {
		t.Fatalf("GetString CommandLine: %v", err)
	}
	if !strings.Contains(cmdLine, "firstboot.bat") {
		t.Fatalf("expected CommandLine to reference firstboot.bat, got: %s", cmdLine)
	}
	pwd, err := hive.GetString(paramsPath, "PWD")
	if err != nil {
		t.Fatalf("GetString PWD: %v", err)
	}
	if !strings.Contains(pwd, "Guestfs") {
		t.Fatalf("expected PWD in Guestfs dir, got: %s", pwd)
	}
}

func TestConfigureCopiesRhsrvanyToGuest(t *testing.T) {
	setupFakeSrvany(t)
	root := t.TempDir()
	hive := regmock.NewMockHive()

	h := version.Classify(&types.InspectData{
		MajorVersion: 10,
		ProductName:  "Windows 10 Enterprise",
	})

	err := firstboot.Configure(&firstboot.Config{
		MountRoot: root,
		Offline:   true,
		Version:   h,
		DriverFiles: []driversource.DriverFile{{
			Name:    "viostor",
			InfPath: "viostor.inf",
		}},
	}, hive, "ControlSet001")
	if err != nil {
		t.Fatalf("Configure returned error: %v", err)
	}

	srvanyPath := filepath.Join(root, "Program Files", "Guestfs", "Firstboot", "rhsrvany.exe")
	data, err := os.ReadFile(srvanyPath)
	if err != nil {
		t.Fatalf("expected rhsrvany.exe in guest Firstboot dir: %v", err)
	}
	if string(data) != "FAKE_EXE" {
		t.Fatalf("unexpected rhsrvany.exe content: %s", data)
	}
}

func TestConfigureLauncherUninstallsService(t *testing.T) {
	setupFakeSrvany(t)
	root := t.TempDir()
	hive := regmock.NewMockHive()

	h := version.Classify(&types.InspectData{
		MajorVersion: 10,
		ProductName:  "Windows Server 2022",
	})

	err := firstboot.Configure(&firstboot.Config{
		MountRoot: root,
		Offline:   true,
		Version:   h,
		DriverFiles: []driversource.DriverFile{{
			Name:    "viostor",
			InfPath: "viostor.inf",
		}},
	}, hive, "ControlSet001")
	if err != nil {
		t.Fatalf("Configure returned error: %v", err)
	}

	bat, err := os.ReadFile(filepath.Join(root, "Program Files", "Guestfs", "Firstboot", "firstboot.bat"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(bat)
	if !strings.Contains(content, "rhsrvany.exe") || !strings.Contains(content, "uninstall") {
		t.Fatalf("expected service uninstall in launcher footer, got: %s", content)
	}
	uninstallIdx := strings.Index(content, "uninstall")
	rebootIdx := strings.Index(content, "shutdown.exe")
	if uninstallIdx > rebootIdx {
		t.Fatalf("service uninstall must come before reboot: %s", content)
	}
}
