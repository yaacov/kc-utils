package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildLUKSSpecClevisTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "key1"), []byte("secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{NbdeClevis: true, LuksDir: dir}
	spec, err := BuildLUKSSpec(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if spec == nil || !spec.Clevis {
		t.Fatalf("got %#v, want Clevis=true (Forklift V2V_NBDE_CLEVIS precedence)", spec)
	}
	if len(spec.KeyFiles) != 0 {
		t.Fatalf("Clevis should skip /etc/luks keyfiles, got %#v", spec.KeyFiles)
	}
}

func TestBuildLUKSSpecKeyFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "key1"), []byte("a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "key2"), []byte("b\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{NbdeClevis: false, LuksDir: dir}
	spec, err := BuildLUKSSpec(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if spec == nil || spec.Clevis || len(spec.KeyFiles) != 2 {
		t.Fatalf("got %#v, want 2 keyfiles", spec)
	}
}

func TestBuildLUKSSpecEmpty(t *testing.T) {
	cfg := &Config{NbdeClevis: false, LuksDir: t.TempDir()}
	spec, err := BuildLUKSSpec(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if spec != nil {
		t.Fatalf("got %#v, want nil", spec)
	}
}

func TestGetEnvBoolNbdeClevis(t *testing.T) {
	t.Setenv(EnvNbdeClevis, "true")
	if !getEnvBool(EnvNbdeClevis, false) {
		t.Fatal("V2V_NBDE_CLEVIS=true should parse as true")
	}
	t.Setenv(EnvNbdeClevis, "1")
	if !getEnvBool(EnvNbdeClevis, false) {
		t.Fatal("V2V_NBDE_CLEVIS=1 should parse as true")
	}
	t.Setenv(EnvNbdeClevis, "false")
	if getEnvBool(EnvNbdeClevis, true) {
		t.Fatal("V2V_NBDE_CLEVIS=false should parse as false")
	}
}
