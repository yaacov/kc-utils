//go:build linux

package guestio_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/guest"
	"github.com/yaacov/kc-utils/pkg/guest/backend"
	"github.com/yaacov/kc-utils/pkg/guest/guestio"

	_ "github.com/yaacov/kc-utils/pkg/guest/plugins/direct"
	_ "github.com/yaacov/kc-utils/pkg/guest/plugins/guestfs"
)

func TestFileHelpersWithActiveDirect(t *testing.T) {
	dir := t.TempDir()
	g, err := guest.AttachMounted(nil, dir, backend.ModeDirect, nil)
	if err != nil {
		t.Fatal(err)
	}
	guest.SetActive(g)
	defer guest.ClearActive()

	p := filepath.Join(dir, "etc", "hostname")
	if err := guestio.FileWrite(p, []byte("guest\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := guestio.FileRead(p)
	if err != nil || string(data) != "guest\n" {
		t.Fatalf("got %q err=%v", data, err)
	}
	free, inodes, err := guestio.FileStatFS(filepath.Join(dir, "etc"))
	if err != nil || free <= 0 || inodes < 0 {
		t.Fatalf("statfs free=%d inodes=%d err=%v", free, inodes, err)
	}
	_ = os.RemoveAll(dir)
}
