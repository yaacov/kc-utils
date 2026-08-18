//go:build unix

package inspect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/registry"
	regmock "github.com/yaacov/kc-utils/pkg/common/registry/mock"
	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestParseOSRelease(t *testing.T) {
	dir := t.TempDir()
	osRelease := filepath.Join(dir, "os-release")
	content := `ID=fedora
PRETTY_NAME="Fedora Linux 39"
VERSION_ID=39
`
	if err := os.WriteFile(osRelease, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	data := &types.InspectData{}
	if err := parseOSRelease(osRelease, data); err != nil {
		t.Fatal(err)
	}
	if data.Distro != "fedora" {
		t.Errorf("distro = %q, want fedora", data.Distro)
	}
	if data.MajorVersion != 39 {
		t.Errorf("major = %d, want 39", data.MajorVersion)
	}
	if data.ProductName != "Fedora Linux 39" {
		t.Errorf("product = %q", data.ProductName)
	}
}

func TestParseOSReleaseWithMinor(t *testing.T) {
	dir := t.TempDir()
	osRelease := filepath.Join(dir, "os-release")
	content := `ID=rhel
VERSION_ID=9.3
PRETTY_NAME="Red Hat Enterprise Linux 9.3"
`
	if err := os.WriteFile(osRelease, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	data := &types.InspectData{}
	if err := parseOSRelease(osRelease, data); err != nil {
		t.Fatal(err)
	}
	if data.MajorVersion != 9 || data.MinorVersion != 3 {
		t.Errorf("version = %d.%d, want 9.3", data.MajorVersion, data.MinorVersion)
	}
}

func TestInspectLinux(t *testing.T) {
	root := t.TempDir()
	etc := filepath.Join(root, "etc")
	if err := os.MkdirAll(etc, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(etc, "os-release"), []byte("ID=ubuntu\nVERSION_ID=22.04\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	data, err := InspectGuest(root)
	if err != nil {
		t.Fatal(err)
	}
	if data.Type != "linux" {
		t.Errorf("type = %q, want linux", data.Type)
	}
	if data.Distro != "ubuntu" {
		t.Errorf("distro = %q, want ubuntu", data.Distro)
	}
}

func TestInspectWindows(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Windows", "System32", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Windows", "System32", "config", "SYSTEM"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Windows", "System32", "config", "SOFTWARE"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	oldOpen := openRegistryHive
	t.Cleanup(func() { openRegistryHive = oldOpen })
	openRegistryHive = func(hivePath string) (registry.Hive, error) {
		hive := regmock.NewMockHive()
		switch filepath.Base(hivePath) {
		case "SOFTWARE":
			hive.CreateKey(`Microsoft\Windows NT\CurrentVersion`)
			hive.SetString(`Microsoft\Windows NT\CurrentVersion`, "ProductName", "Windows Server 2022")
			hive.SetDWORD(`Microsoft\Windows NT\CurrentVersion`, "CurrentMajorVersionNumber", 10)
			hive.SetDWORD(`Microsoft\Windows NT\CurrentVersion`, "CurrentMinorVersionNumber", 0)
		case "SYSTEM":
			hive.CreateKey(`Select`)
			hive.SetDWORD(`Select`, "Current", 2)
		}
		return hive, nil
	}

	data, err := InspectGuest(root)
	if err != nil {
		t.Fatal(err)
	}
	if data.Type != "windows" {
		t.Errorf("type = %q, want windows", data.Type)
	}
	if data.ProductName != "Windows Server 2022" {
		t.Errorf("product = %q, want Windows Server 2022", data.ProductName)
	}
	if data.MajorVersion != 10 || data.MinorVersion != 0 {
		t.Errorf("version = %d.%d, want 10.0", data.MajorVersion, data.MinorVersion)
	}
}

func TestInspectWindowsMetadata(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "WINDOWS", "System32", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"SYSTEM", "SOFTWARE"} {
		if err := os.WriteFile(filepath.Join(root, "WINDOWS", "System32", "config", name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	oldOpen := openRegistryHive
	t.Cleanup(func() { openRegistryHive = oldOpen })
	openRegistryHive = func(hivePath string) (registry.Hive, error) {
		hive := regmock.NewMockHive()
		if filepath.Base(hivePath) == "SYSTEM" {
			hive.CreateKey(`Select`)
			hive.SetDWORD(`Select`, "Current", 5)
		}
		return hive, nil
	}

	got, err := InspectWindowsMetadata(root)
	if err != nil {
		t.Fatalf("InspectWindowsMetadata error: %v", err)
	}
	if got.SystemRoot != "WINDOWS" {
		t.Errorf("SystemRoot = %q, want WINDOWS", got.SystemRoot)
	}
	if got.SystemHive != filepath.Join("WINDOWS", "System32", "config", "SYSTEM") {
		t.Errorf("SystemHive = %q", got.SystemHive)
	}
	if got.SoftwareHive != filepath.Join("WINDOWS", "System32", "config", "SOFTWARE") {
		t.Errorf("SoftwareHive = %q", got.SoftwareHive)
	}
	if got.CurrentControlSet != 5 {
		t.Errorf("CurrentControlSet = %d, want 5", got.CurrentControlSet)
	}
}

func TestDetectArch(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "lib", "modules", "5.14.0-284.el9.x86_64"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := DetectArch(root); got != "x86_64" {
		t.Errorf("arch = %q, want x86_64", got)
	}
}
