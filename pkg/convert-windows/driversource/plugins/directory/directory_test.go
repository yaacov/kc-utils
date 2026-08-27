package directory

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/convert-windows/driversource"
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

func TestFindDriversWin2008FallbackTo2k8R2(t *testing.T) {
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
	drivers, err := src.FindDrivers("x86_64", "Windows Server 2008 Enterprise", h.DriverOSPreferences(), h.DriverOSFallbacks())
	if err != nil {
		t.Fatal(err)
	}
	if len(drivers) == 0 {
		t.Fatal("expected drivers from 2k8R2 fallback")
	}
	if filepath.Base(filepath.Dir(drivers[0].InfPath)) != "2k8R2" {
		t.Fatalf("expected 2k8R2 dir, got %q", drivers[0].SrcPath)
	}
}

func TestConfigure(t *testing.T) {
	src, ok := driversource.Sources.Get("directory")
	if !ok {
		t.Fatal("directory source not registered")
	}
	d, ok := src.(*DirectorySource)
	if !ok {
		t.Fatalf("unexpected source type %T", src)
	}
	origBase, origGA := d.BasePath, d.GuestAgentDir
	t.Cleanup(func() {
		d.BasePath = origBase
		d.GuestAgentDir = origGA
	})

	Configure("/tmp/virtio-win")
	if got := d.basePath(); got != "/tmp/virtio-win/drivers/by-os" {
		t.Fatalf("basePath = %q", got)
	}
	if got := d.guestAgentDir(); got != "/tmp/virtio-win/guest-agent" {
		t.Fatalf("guestAgentDir = %q", got)
	}
}
