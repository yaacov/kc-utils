package driversource

import (
	"errors"
	"os"
	"path/filepath"
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
func (s *stubSource) FindDrivers(arch, osVersion string, osPrefs, osFallbacks []string) ([]DriverFile, error) {
	s.calls++
	return s.files, s.err
}

func TestCollectDriversDirectoryOnly(t *testing.T) {
	orig := Sources
	Sources = plugin.NewRegistry[string, DriverSource]()
	t.Cleanup(func() { Sources = orig })

	srcPath := t.TempDir()
	infPath := filepath.Join(srcPath, "viostor.inf")
	if err := os.WriteFile(infPath, []byte("[Version]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	dir := &stubSource{
		available: true,
		files:     []DriverFile{{Name: "viostor", SrcPath: srcPath, InfPath: infPath}},
	}
	Sources.Register("directory", dir)

	files, err := CollectDrivers("x86_64", "Windows Server 2019", "win10", []string{"w10"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].SrcPath != srcPath {
		t.Fatalf("got %#v", files)
	}
	if len(files[0].Files) != 1 || files[0].Files[0] != infPath {
		t.Fatalf("Files=%v", files[0].Files)
	}
}

func TestCollectDriversRequiresDirectory(t *testing.T) {
	orig := Sources
	Sources = plugin.NewRegistry[string, DriverSource]()
	t.Cleanup(func() { Sources = orig })

	Sources.Register("directory", &stubSource{available: false})
	if _, err := CollectDrivers("x86_64", "Windows Server 2019", "winunknown", nil, nil); err == nil {
		t.Fatal("expected error when directory unavailable")
	}
}

func TestCollectDriversPropagatesFindError(t *testing.T) {
	orig := Sources
	Sources = plugin.NewRegistry[string, DriverSource]()
	t.Cleanup(func() { Sources = orig })

	Sources.Register("directory", &stubSource{available: true, err: errors.New("boom")})
	if _, err := CollectDrivers("x86_64", "w10", "win10", nil, nil); err == nil {
		t.Fatal("expected error from FindDrivers")
	}
}
