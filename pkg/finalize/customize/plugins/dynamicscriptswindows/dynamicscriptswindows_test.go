//go:build linux

package dynamicscriptswindows

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanScripts(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "20_win_firstboot_join-domain.ps1"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	scripts, err := scanScripts(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(scripts) != 1 {
		t.Fatalf("got %d scripts, want 1", len(scripts))
	}
	if scripts[0].Priority != 20 {
		t.Errorf("script = %+v, want priority 20", scripts[0])
	}
}
