package driversource_test

import (
	"errors"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/plugin"
	"github.com/yaacov/kc-utils/pkg/convert-windows/driversource"
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
	if _, err := driversource.CollectDrivers("x86_64", "Windows Server 2008", h.DriverOSPreferences(), h.DriverOSFallbacks()); err != nil {
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
	if _, err := driversource.CollectDrivers("x86_64", "w10", nil, nil); err == nil {
		t.Fatal("expected error from FindDrivers")
	}
}
