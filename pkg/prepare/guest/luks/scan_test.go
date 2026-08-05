package luks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanKeyFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "key1"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "key2"), []byte("secret2"), 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := ScanKeyFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2", len(files))
	}
}

func TestKeyFilesMap(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "disk.key"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := KeyFilesMap(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 1 {
		t.Fatalf("got %d entries, want 1", len(m))
	}
	if _, ok := m["all"]; !ok {
		t.Errorf("expected all key, got %v", m)
	}
}
