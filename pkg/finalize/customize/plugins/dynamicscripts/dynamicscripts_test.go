//go:build linux

package dynamicscripts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanScriptsLinux(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"10_linux_firstboot_config.sh", "05_linux_run_install.sh", "bad.sh"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/bash\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	scripts, err := scanScripts(dir, "linux")
	if err != nil {
		t.Fatal(err)
	}
	if len(scripts) != 2 {
		t.Fatalf("got %d scripts, want 2", len(scripts))
	}
	if scripts[0].Priority != 5 || scripts[0].Action != "run" {
		t.Errorf("first script = %+v, want priority 5 run", scripts[0])
	}
	if scripts[1].Priority != 10 || scripts[1].Action != "firstboot" {
		t.Errorf("second script = %+v, want priority 10 firstboot", scripts[1])
	}
}

func TestScanScriptsWindows(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "20_win_firstboot_join-domain.ps1"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	scripts, err := scanScripts(dir, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(scripts) != 1 || scripts[0].Action != "win-firstboot" {
		t.Fatalf("got %+v, want one win-firstboot script", scripts)
	}
}
