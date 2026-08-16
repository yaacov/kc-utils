//go:build linux

package guest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/guest"

	_ "github.com/yaacov/kc-utils/pkg/guest/direct"
	_ "github.com/yaacov/kc-utils/pkg/guest/guestfs"
)

func TestFileHelpersWithActiveDirect(t *testing.T) {
	dir := t.TempDir()
	g, err := guest.AttachMounted(nil, dir, guest.ModeDirect, nil)
	if err != nil {
		t.Fatal(err)
	}
	guest.SetActive(g)
	defer guest.ClearActive()

	p := filepath.Join(dir, "etc", "hostname")
	if err := guest.FileWrite(p, []byte("guest\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := guest.FileRead(p)
	if err != nil || string(data) != "guest\n" {
		t.Fatalf("got %q err=%v", data, err)
	}
	free, inodes, err := guest.FileStatFS(filepath.Join(dir, "etc"))
	if err != nil || free <= 0 || inodes < 0 {
		t.Fatalf("statfs free=%d inodes=%d err=%v", free, inodes, err)
	}
	_ = os.RemoveAll(dir)
}
