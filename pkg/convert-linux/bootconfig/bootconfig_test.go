package bootconfig

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

type fakeBootloader struct {
	added   []string
	removed []string
}

func (f *fakeBootloader) Detect(string) bool                      { return true }
func (f *fakeBootloader) GetDefaultKernel(string) (string, error) { return "", nil }
func (f *fakeBootloader) SetDefaultKernel(string, string) error   { return nil }
func (f *fakeBootloader) AddKernelArg(_ string, arg string) error {
	f.added = append(f.added, arg)
	return nil
}
func (f *fakeBootloader) RemoveKernelArg(_ string, prefix string) error {
	f.removed = append(f.removed, prefix)
	return nil
}
func (f *fakeBootloader) RegenerateConfig(string) error { return nil }

type fakeDistro struct {
	console string
}

func (d *fakeDistro) Matches(*types.InspectData) bool { return true }
func (d *fakeDistro) DefaultKernelArgs() []string     { return nil }
func (d *fakeDistro) DefaultConsole() string          { return d.console }

func TestConfigureConsoleNilHandler(t *testing.T) {
	ConfigureConsole("/guest", nil, &fakeDistro{console: "ttyS0"})
}

func TestConfigureConsole(t *testing.T) {
	bl := &fakeBootloader{}
	ConfigureConsole("/guest", bl, &fakeDistro{console: "ttyAMA0"})

	if !contains(bl.removed, "rhgb") || !contains(bl.removed, "quiet") {
		t.Errorf("removed = %v, want rhgb and quiet", bl.removed)
	}
	if !contains(bl.added, "console=ttyAMA0") {
		t.Errorf("added = %v, want console=ttyAMA0", bl.added)
	}
}

func TestConfigureConsoleDefaultDistro(t *testing.T) {
	bl := &fakeBootloader{}
	ConfigureConsole("/guest", bl, nil)
	if !contains(bl.added, "console=ttyS0") {
		t.Errorf("added = %v, want console=ttyS0", bl.added)
	}
}

func TestConfigureDisplayNilHandler(t *testing.T) {
	ConfigureDisplay("/guest", nil)
}

func TestConfigureDisplay(t *testing.T) {
	bl := &fakeBootloader{}
	ConfigureDisplay("/guest", bl)

	if !contains(bl.removed, "vga") || !contains(bl.removed, "video=cirrus") {
		t.Errorf("removed = %v, want vga and video=cirrus", bl.removed)
	}
	if !contains(bl.added, "video=virtio") {
		t.Errorf("added = %v, want video=virtio", bl.added)
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
