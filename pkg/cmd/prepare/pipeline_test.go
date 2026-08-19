//go:build unix

package prepare

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestResolveInputDisksFromDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "disk0.img")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	in := types.PrepareInput{DiskDir: dir}
	if err := resolveInputDisks(&in); err != nil {
		t.Fatal(err)
	}
	if len(in.Disks) != 1 || in.Disks[0].Path != path || in.Disks[0].Format != "raw" {
		t.Fatalf("disks = %+v", in.Disks)
	}
}

func TestResolveInputDisksKeepsExplicitList(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "disk0.img"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	in := types.PrepareInput{
		DiskDir: dir,
		Disks:   []types.DiskSpec{{Path: "/dev/block0", Format: "raw"}},
	}
	if err := resolveInputDisks(&in); err != nil {
		t.Fatal(err)
	}
	if len(in.Disks) != 1 || in.Disks[0].Path != "/dev/block0" {
		t.Fatalf("explicit disks should win, got %+v", in.Disks)
	}
}

func TestResolveInputDisksNoDir(t *testing.T) {
	in := types.PrepareInput{}
	if err := resolveInputDisks(&in); err != nil {
		t.Fatal(err)
	}
	if len(in.Disks) != 0 {
		t.Fatalf("disks = %+v", in.Disks)
	}
}
