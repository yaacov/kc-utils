package driversource_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/plugin"
	"github.com/yaacov/kc-utils/pkg/convert-windows/driversource"
	"github.com/yaacov/kc-utils/pkg/convert-windows/driversource/plugins/directory"
	"github.com/yaacov/kc-utils/pkg/convert-windows/version"
)

type stubSource struct {
	available bool
	files     []driversource.DriverFile
	err       error
	lastPrefs []string
	lastFB    []string
}

func (s *stubSource) Available() bool { return s.available }
func (s *stubSource) FindDrivers(arch, osVersion string, osPrefs, osFallbacks []string) ([]driversource.DriverFile, error) {
	s.lastPrefs = osPrefs
	s.lastFB = osFallbacks
	return s.files, s.err
}

func TestCollectDriversPassesHandlerPrefs(t *testing.T) {
	orig := driversource.Sources
	driversource.Sources = plugin.NewRegistry[string, driversource.DriverSource]()
	t.Cleanup(func() { driversource.Sources = orig })

	dir := &stubSource{available: true, files: []driversource.DriverFile{{Name: "viostor"}}}
	driversource.Sources.Register("directory", dir)

	h, ok := version.Handlers.Get("win2008")
	if !ok {
		t.Fatal("win2008 handler not registered")
	}
	if _, err := driversource.CollectDrivers("x86_64", "Windows Server 2008", "win2008", h.DriverOSPreferences(), h.DriverOSFallbacks()); err != nil {
		t.Fatal(err)
	}
	if len(dir.lastPrefs) == 0 || dir.lastPrefs[0] != "2k8" {
		t.Fatalf("prefs = %v, want 2k8 first", dir.lastPrefs)
	}
}

func TestCollectDriversPropagatesFindError(t *testing.T) {
	orig := driversource.Sources
	driversource.Sources = plugin.NewRegistry[string, driversource.DriverSource]()
	t.Cleanup(func() { driversource.Sources = orig })

	driversource.Sources.Register("directory", &stubSource{available: true, err: errors.New("boom")})
	if _, err := driversource.CollectDrivers("x86_64", "w10", "win10", nil, nil); err == nil {
		t.Fatal("expected error from FindDrivers")
	}
}

func TestCollectDriversPreWin8MissingDirHint(t *testing.T) {
	base := t.TempDir()
	amd64 := filepath.Join(base, "amd64")
	if err := os.MkdirAll(filepath.Join(amd64, "2k8R2"), 0o755); err != nil {
		t.Fatal(err)
	}

	orig := driversource.Sources
	driversource.Sources = plugin.NewRegistry[string, driversource.DriverSource]()
	t.Cleanup(func() { driversource.Sources = orig })

	driversource.Sources.Register("directory", &directory.DirectorySource{BasePath: base})

	_, err := driversource.CollectDrivers("x86_64", "Windows Server 2008", "win2008", []string{"2k8"}, nil)
	if err == nil {
		t.Fatal("expected error when only 2k8R2 is present")
	}
	msg := err.Error()
	for _, want := range []string{"2k8", "win2008", "build/kc-v2v/vendor/README.md"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q missing %q", msg, want)
		}
	}
}

func TestCollectDriversOmitsGuestAgentForWin2008(t *testing.T) {
	base := t.TempDir()
	osDir := filepath.Join(base, "amd64", "2k8")
	if err := os.MkdirAll(osDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(osDir, "viostor.inf"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	gaDir := filepath.Join(base, "guest-agent")
	if err := os.MkdirAll(gaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gaDir, "qemu-ga-x86_64.msi"), []byte("msi"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig := driversource.Sources
	driversource.Sources = plugin.NewRegistry[string, driversource.DriverSource]()
	t.Cleanup(func() { driversource.Sources = orig })

	driversource.Sources.Register("directory", &directory.DirectorySource{
		BasePath:      base,
		GuestAgentDir: gaDir,
	})

	files, err := driversource.CollectDrivers("x86_64", "Windows Server 2008", "win2008", []string{"2k8"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f.Name == "qemu-ga" {
			t.Fatal("win2008 should not collect qemu-ga MSI")
		}
	}
}
