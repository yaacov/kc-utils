package driversource

import (
	"errors"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/plugin"
)

type stubSource struct {
	available bool
	files     []DriverFile
	err       error
	calls     int
}

func (s *stubSource) Available() bool { return s.available }
func (s *stubSource) FindDrivers(arch, osVersion string) ([]DriverFile, error) {
	s.calls++
	return s.files, s.err
}

func TestCollectDriversISOOnly(t *testing.T) {
	orig := Sources
	Sources = plugin.NewRegistry[string, DriverSource]()
	t.Cleanup(func() { Sources = orig })

	iso := &stubSource{
		available: true,
		files:     []DriverFile{{Name: "viostor", SrcPath: "/iso/viostor"}},
	}
	Sources.Register("iso", iso)

	files, _, err := CollectDrivers("x86_64", "Windows Server 2019")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].SrcPath != "/iso/viostor" {
		t.Fatalf("got %#v", files)
	}
}

func TestCollectDriversRequiresISO(t *testing.T) {
	orig := Sources
	Sources = plugin.NewRegistry[string, DriverSource]()
	t.Cleanup(func() { Sources = orig })

	Sources.Register("iso", &stubSource{available: false})
	if _, _, err := CollectDrivers("x86_64", "Windows Server 2019"); err == nil {
		t.Fatal("expected error when ISO unavailable")
	}
}

func TestCollectDriversPropagatesFindError(t *testing.T) {
	orig := Sources
	Sources = plugin.NewRegistry[string, DriverSource]()
	t.Cleanup(func() { Sources = orig })

	Sources.Register("iso", &stubSource{available: true, err: errors.New("boom")})
	if _, _, err := CollectDrivers("x86_64", "w10"); err == nil {
		t.Fatal("expected error from FindDrivers")
	}
}
