package directory

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/convert-windows/version"
)

func TestFindDriversByOSLayout(t *testing.T) {
	root := t.TempDir()
	osDir := filepath.Join(root, "amd64", "w10")
	if err := os.MkdirAll(osDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"viostor.inf", "viostor.sys", "netkvm.inf"} {
		if err := os.WriteFile(filepath.Join(osDir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gaDir := filepath.Join(root, "guest-agent")
	if err := os.MkdirAll(gaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gaDir, "qemu-ga-x86_64.msi"), []byte("msi"), 0o644); err != nil {
		t.Fatal(err)
	}

	src := &DirectorySource{BasePath: root, GuestAgentDir: gaDir}
	drivers, err := src.FindDrivers("x86_64", "Windows 10", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(drivers) < 3 {
		t.Fatalf("expected at least 3 drivers, got %d", len(drivers))
	}

	var hasQEMU bool
	for _, d := range drivers {
		if d.Name == "qemu-ga" {
			hasQEMU = true
		}
		if d.SrcPath != osDir && d.Name != "qemu-ga" {
			t.Fatalf("unexpected SrcPath %q for %q", d.SrcPath, d.Name)
		}
	}
	if !hasQEMU {
		t.Fatal("expected qemu-ga driver")
	}
}

func TestFindDriversWin2008Prefers2k8Dir(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"2k8", "2k8R2"} {
		osDir := filepath.Join(root, "amd64", dir)
		if err := os.MkdirAll(osDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(osDir, "viostor.inf"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	h, ok := version.Handlers.Get("win2008")
	if !ok {
		t.Fatal("win2008 handler missing")
	}

	src := &DirectorySource{BasePath: root}
	drivers, err := src.FindDrivers("x86_64", "Windows Server (R) 2008 Enterprise\x00", h.DriverOSPreferences(), h.DriverOSFallbacks())
	if err != nil {
		t.Fatal(err)
	}
	if len(drivers) == 0 {
		t.Fatal("expected drivers")
	}
	if filepath.Base(filepath.Dir(drivers[0].InfPath)) != "2k8" {
		t.Fatalf("expected 2k8 dir, got %q", drivers[0].SrcPath)
	}
}

func TestFindDriversWin2008Requires2k8Dir(t *testing.T) {
	root := t.TempDir()
	osDir := filepath.Join(root, "amd64", "2k8R2")
	if err := os.MkdirAll(osDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(osDir, "viostor.inf"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	h, ok := version.Handlers.Get("win2008")
	if !ok {
		t.Fatal("win2008 handler missing")
	}

	src := &DirectorySource{BasePath: root}
	if _, err := src.FindDrivers("x86_64", "Windows Server 2008 Enterprise", h.DriverOSPreferences(), h.DriverOSFallbacks()); err == nil {
		t.Fatal("expected error when 2k8 dir is missing")
	}
}
