//go:build unix

package core

import (
	"os"
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/guest/runtime"
)

// fakeRuntime is a scripted runtime for exercising core logic without touching
// real block devices. Only the methods used by the tests are meaningful.
type fakeRuntime struct {
	// runFn maps a command (argv joined by space) to a canned result.
	runFn func(argv []string) runtime.CommandResult
	files map[string][]byte
}

func (f *fakeRuntime) Run(spec *runtime.CommandSpec) (runtime.CommandResult, error) {
	if f.runFn == nil {
		return runtime.CommandResult{}, nil
	}
	return f.runFn(spec.Argv), nil
}

func (f *fakeRuntime) ReadFile(path string) ([]byte, error) {
	if d, ok := f.files[path]; ok {
		return d, nil
	}
	return nil, os.ErrNotExist
}
func (f *fakeRuntime) WriteFile(path string, data []byte, _ os.FileMode) error {
	if f.files == nil {
		f.files = map[string][]byte{}
	}
	f.files[path] = data
	return nil
}
func (f *fakeRuntime) MkdirAll(string, os.FileMode) error    { return nil }
func (f *fakeRuntime) Remove(string) error                   { return nil }
func (f *fakeRuntime) RemoveAll(string) error                { return nil }
func (f *fakeRuntime) Rename(string, string) error           { return nil }
func (f *fakeRuntime) Symlink(string, string) error          { return nil }
func (f *fakeRuntime) Readlink(string) (string, error)       { return "", nil }
func (f *fakeRuntime) Chmod(string, os.FileMode) error       { return nil }
func (f *fakeRuntime) Stat(string) (runtime.FileInfo, error) { return runtime.FileInfo{}, nil }
func (f *fakeRuntime) ReadDir(string) ([]runtime.DirEntry, error) {
	return nil, nil
}
func (f *fakeRuntime) Glob(string) ([]string, error)                 { return nil, nil }
func (f *fakeRuntime) DeviceRead(string, int64, int) ([]byte, error) { return nil, nil }
func (f *fakeRuntime) DeviceWrite(string, int64, []byte) error       { return nil }
func (f *fakeRuntime) StatFS(string) (int64, int64, error)           { return 0, 0, nil }
func (f *fakeRuntime) Close() error                                  { return nil }

func TestNormalizeFSType(t *testing.T) {
	cases := map[string]string{
		"ext4":        "ext4",
		"xfs":         "xfs",
		"ntfs":        "ntfs3",
		"NTFS":        "ntfs3",
		" ntfs ":      "ntfs3",
		"":            "",
		"  ":          "",
		"crypto_LUKS": "crypto_LUKS",
		"LVM2_member": "LVM2_member",
		"swap":        "swap",
	}
	for in, want := range cases {
		if got := normalizeFSType(in); got != want {
			t.Errorf("normalizeFSType(%q) = %q, want %q", in, got, want)
		}
	}
}

const sampleLsblk = `{
  "blockdevices": [
    {"name":"vda","path":"/dev/vda","type":"disk","fstype":null,"serial":"kc-disk-0",
      "children":[
        {"name":"vda1","path":"/dev/vda1","type":"part","fstype":"ext4"},
        {"name":"vda2","path":"/dev/vda2","type":"part","fstype":"LVM2_member"}
      ]}
  ]
}`

func TestDiscoverDevice(t *testing.T) {
	rt := &fakeRuntime{runFn: func(argv []string) runtime.CommandResult {
		if len(argv) > 0 && argv[0] == "lsblk" {
			return runtime.CommandResult{Stdout: []byte(sampleLsblk)}
		}
		return runtime.CommandResult{}
	}}
	b := New(rt, true)
	if err := b.DiscoverDevice("/dev/vda", "/images/disk0.qcow2", "qcow2"); err != nil {
		t.Fatalf("DiscoverDevice: %v", err)
	}
	disks := b.DiskInfos()
	if len(disks) != 1 {
		t.Fatalf("want 1 disk, got %d", len(disks))
	}
	di := disks[0]
	if di.Path != "/images/disk0.qcow2" || di.Format != "qcow2" {
		t.Errorf("disk meta wrong: %+v", di)
	}
	if len(di.Partitions) != 2 {
		t.Fatalf("want 2 partitions, got %d", len(di.Partitions))
	}
	if di.Partitions[0].DevicePath != "/dev/vda1" || di.Partitions[0].FSType != "ext4" {
		t.Errorf("part0 wrong: %+v", di.Partitions[0])
	}
	if di.Partitions[0].Index != 1 || di.Partitions[1].Index != 2 {
		t.Errorf("partition indexes not 1-based sequential: %+v", di.Partitions)
	}
	if di.Partitions[1].FSType != "LVM2_member" {
		t.Errorf("part1 fstype wrong: %q", di.Partitions[1].FSType)
	}
	if want := []string{"/dev/vda1", "/dev/vda2"}; strings.Join(b.partDevices, ",") != strings.Join(want, ",") {
		t.Errorf("partDevices = %v, want %v", b.partDevices, want)
	}
}

func TestListBlockDevices(t *testing.T) {
	rt := &fakeRuntime{runFn: func(argv []string) runtime.CommandResult {
		return runtime.CommandResult{Stdout: []byte(sampleLsblk)}
	}}
	b := New(rt, false)
	disks, err := b.ListBlockDevices()
	if err != nil {
		t.Fatalf("ListBlockDevices: %v", err)
	}
	if len(disks) != 1 || disks[0].Serial != "kc-disk-0" {
		t.Fatalf("unexpected disks: %+v", disks)
	}
	if len(disks[0].Children) != 2 {
		t.Errorf("want 2 children, got %d", len(disks[0].Children))
	}
}
