package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildBitLockerSpecKeyFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pass1"), []byte("secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	spec, err := BuildBitLockerSpec(&Config{BitLockerDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if spec == nil || len(spec.KeyFiles) != 1 {
		t.Fatalf("spec = %+v", spec)
	}
}

func TestBuildBitLockerSpecEmpty(t *testing.T) {
	spec, err := BuildBitLockerSpec(&Config{BitLockerDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if spec != nil {
		t.Fatalf("expected nil spec, got %+v", spec)
	}
}
