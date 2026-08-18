//go:build unix

package inspect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRecordIncludesBootAndEFIWhenPresent(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"boot", filepath.Join("boot", "efi")} {
		if err := os.MkdirAll(filepath.Join(root, rel), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	got := Record(root)
	if len(got) != 3 {
		t.Fatalf("Record returned %d entries, want 3", len(got))
	}
	if got[0].Path != "/" || got[1].Path != "/boot" || got[2].Path != "/boot/efi" {
		t.Fatalf("unexpected paths: %+v", got)
	}
}
