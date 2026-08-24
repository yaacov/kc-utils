package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func TestLinkCertificatesSkipsNonVsphere(t *testing.T) {
	if err := LinkCertificates(&Config{Source: "ec2"}); err != nil {
		t.Fatal(err)
	}
}

func TestLinkCertificatesNoOpWithoutSecret(t *testing.T) {
	if ProviderCACertMounted() {
		t.Skip("provider CA secret mounted")
	}
	if err := LinkCertificates(&Config{Source: "vSphere"}); err != nil {
		t.Fatal(err)
	}
}

func TestProviderCACertMountedAbsent(t *testing.T) {
	if _, err := os.Stat(DefaultCaCert); err == nil {
		t.Skip("provider CA secret mounted")
	}
	if ProviderCACertMounted() {
		t.Fatal("expected false when secret is absent")
	}
}

func TestBitLockerDirFromEnv(t *testing.T) {
	t.Setenv(EnvBitLockerDir, "/env/bitlocker")
	if got := envOr(EnvBitLockerDir, DefaultBitLockerDir); got != "/env/bitlocker" {
		t.Fatalf("envOr = %q, want /env/bitlocker", got)
	}
}

func TestBuildPrepareInputBitLockerDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "recovery"), []byte("secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{BitLockerDir: dir}
	in, err := BuildPrepareInput(cfg, nil, types.SourceSpec{})
	if err != nil {
		t.Fatal(err)
	}
	if in.BitLocker == nil || len(in.BitLocker.KeyFiles) != 1 {
		t.Fatalf("BitLocker = %+v", in.BitLocker)
	}
}
