package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinkCertificatesIdempotent(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "cacert")
	system := filepath.Join(dir, "ca-bundle.crt.bak")
	dest := filepath.Join(dir, "ca-bundle.crt")

	if err := os.WriteFile(system, []byte("system-ca"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		Source:         "vSphere",
		CaBundle:       dest,
		CaCert:         secret,
		SystemCaBundle: system,
	}
	if err := LinkCertificates(cfg); err != nil {
		t.Fatalf("first link: %v", err)
	}
	if err := LinkCertificates(cfg); err != nil {
		t.Fatalf("second link (idempotent): %v", err)
	}

	got, err := os.Readlink(dest)
	if err != nil {
		t.Fatal(err)
	}
	if got != system {
		t.Fatalf("symlink target = %q, want %q", got, system)
	}
}

func TestLinkCertificatesPrefersSecret(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "cacert")
	system := filepath.Join(dir, "ca-bundle.crt.bak")
	dest := filepath.Join(dir, "ca-bundle.crt")

	if err := os.WriteFile(secret, []byte("secret-ca"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(system, []byte("system-ca"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		Source:         "vSphere",
		CaBundle:       dest,
		CaCert:         secret,
		SystemCaBundle: system,
	}
	if err := LinkCertificates(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := os.Readlink(dest)
	if err != nil {
		t.Fatal(err)
	}
	if got != secret {
		t.Fatalf("symlink target = %q, want %q", got, secret)
	}
}

func TestResolveCaPathsDefaults(t *testing.T) {
	cfg := &Config{}
	resolveCaPaths(cfg)
	if cfg.CaBundle != DefaultCaBundle {
		t.Fatalf("CaBundle = %q, want %q", cfg.CaBundle, DefaultCaBundle)
	}
	if cfg.CaCert != DefaultCaCert {
		t.Fatalf("CaCert = %q, want %q", cfg.CaCert, DefaultCaCert)
	}
	if cfg.SystemCaBundle != DefaultSystemCaBundle {
		t.Fatalf("SystemCaBundle = %q, want %q", cfg.SystemCaBundle, DefaultSystemCaBundle)
	}
}

func TestResolveCaPathsFromEnv(t *testing.T) {
	t.Setenv(EnvCaBundle, "/custom/bundle")
	t.Setenv(EnvCaCert, "/custom/cacert")
	t.Setenv(EnvSystemCaBundle, "/custom/system")

	cfg := &Config{}
	resolveCaPaths(cfg)
	if cfg.CaBundle != "/custom/bundle" || cfg.CaCert != "/custom/cacert" || cfg.SystemCaBundle != "/custom/system" {
		t.Fatalf("unexpected paths: %+v", cfg)
	}
}
