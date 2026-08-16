//go:build linux

package guest

import (
	"os"
	"path/filepath"
	"testing"
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
	_ = os.RemoveAll(dir)
}
