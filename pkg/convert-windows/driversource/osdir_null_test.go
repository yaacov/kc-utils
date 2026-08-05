package driversource

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindBestOSDirWin2008ProductNameWithNull(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "2k8R2"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := FindBestOSDir(base, "Windows Server (R) 2008 Enterprise\x00")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "2k8R2" {
		t.Fatalf("got %q, want 2k8R2", got)
	}
}
