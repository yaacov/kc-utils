//go:build linux

package guest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/backend/plugins/direct"
)

func TestFileHelpersWithoutActive(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f")
	if err := FileWrite(p, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := FileRead(p)
	if err != nil || string(data) != "hi" {
		t.Fatalf("read: %q err=%v", data, err)
	}
	if !FileExists(p) || FileIsDir(p) {
		t.Fatal("exists/isdir")
	}
}

func TestFileHelpersWithActiveDirect(t *testing.T) {
	dir := t.TempDir()
	g := &Guest{rootPath: dir, backendName: BackendDirect, backend: direct.NewMounted(nil, dir, nil)}
	SetActive(g)
	defer ClearActive()

	p := filepath.Join(dir, "etc", "hostname")
	if err := FileWrite(p, []byte("guest\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := FileRead(p)
	if err != nil || string(data) != "guest\n" {
		t.Fatalf("got %q err=%v", data, err)
	}
	free, inodes, err := FileStatFS(filepath.Join(dir, "etc"))
	if err != nil || free <= 0 || inodes < 0 {
		t.Fatalf("statfs free=%d inodes=%d err=%v", free, inodes, err)
	}
	_ = os.RemoveAll(dir)
}
