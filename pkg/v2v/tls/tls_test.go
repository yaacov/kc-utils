package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yaacov/kc-utils/pkg/v2v/config"
)

func writeTestCABundle(t *testing.T, dir string) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-ca"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IsCA:         true,
		KeyUsage:     x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(dir, "ca-bundle.crt")
	if err := os.WriteFile(bundle, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		t.Fatal(err)
	}
	return bundle
}

func TestForkliftTLSInsecure(t *testing.T) {
	policy, err := ForkliftTLS(true)
	if err != nil {
		t.Fatal(err)
	}
	if policy.Mode != ModeInsecure {
		t.Fatalf("Mode = %v, want ModeInsecure", policy.Mode)
	}
}

func TestForkliftTLSSystemCA(t *testing.T) {
	if _, err := os.Stat(config.DefaultCaCert); err == nil {
		t.Skip("provider CA secret mounted")
	}
	policy, err := ForkliftTLS(false)
	if err != nil {
		t.Fatal(err)
	}
	if policy.Mode != ModeSystemCA {
		t.Fatalf("policy = %+v", policy)
	}
}

func TestCopyTLSInsecure(t *testing.T) {
	policy, err := CopyTLS(true, "")
	if err != nil {
		t.Fatal(err)
	}
	if policy.Mode != ModeInsecure {
		t.Fatalf("Mode = %v, want ModeInsecure", policy.Mode)
	}
}

func TestCopyTLSCustomCA(t *testing.T) {
	bundle := writeTestCABundle(t, t.TempDir())
	policy, err := CopyTLS(false, bundle)
	if err != nil {
		t.Fatal(err)
	}
	if policy.Mode != ModeCustomCA || policy.CaBundlePath != bundle {
		t.Fatalf("policy = %+v", policy)
	}
}

func TestCopyTLSSystemCA(t *testing.T) {
	policy, err := CopyTLS(false, "")
	if err != nil {
		t.Fatal(err)
	}
	if policy.Mode != ModeSystemCA {
		t.Fatalf("policy = %+v", policy)
	}
}

func TestCopyTLSMissingCAFile(t *testing.T) {
	_, err := CopyTLS(false, filepath.Join(t.TempDir(), "missing.crt"))
	if err == nil {
		t.Fatal("expected error for missing CA cert file")
	}
}

func TestVCenterConfigInsecure(t *testing.T) {
	cfg, err := VCenterConfig(Policy{Mode: ModeInsecure})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.InsecureSkipVerify {
		t.Fatal("expected InsecureSkipVerify=true")
	}
}

func TestVCenterConfigSystemCA(t *testing.T) {
	cfg, err := VCenterConfig(Policy{Mode: ModeSystemCA})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.InsecureSkipVerify || cfg.RootCAs != nil {
		t.Fatal("expected nil RootCAs for system CA mode")
	}
}

func TestVCenterConfigValidBundle(t *testing.T) {
	bundle := writeTestCABundle(t, t.TempDir())
	cfg, err := VCenterConfig(Policy{Mode: ModeCustomCA, CaBundlePath: bundle})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.InsecureSkipVerify || cfg.RootCAs == nil {
		t.Fatal("expected verified TLS with RootCAs")
	}
}

func TestVCenterConfigInvalidPEM(t *testing.T) {
	dir := t.TempDir()
	bundle := filepath.Join(dir, "ca-bundle.crt")
	if err := os.WriteFile(bundle, []byte("not a pem file"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := VCenterConfig(Policy{Mode: ModeCustomCA, CaBundlePath: bundle})
	if err == nil {
		t.Fatal("expected error for invalid PEM")
	}
}

func TestInsecureFromLibvirtURL(t *testing.T) {
	if !InsecureFromLibvirtURL("vpx://user@vcenter/dc/host?no_verify=1") {
		t.Fatal("expected insecure")
	}
	if InsecureFromLibvirtURL("vpx://user@vcenter/dc/host?cacert=/opt/ca-bundle.crt") {
		t.Fatal("expected secure")
	}
	if InsecureFromLibvirtURL("vpx://user@vcenter/dc/host?no_verify=0") {
		t.Fatal("no_verify=0 should not enable insecure")
	}
}

func TestInsecureFromQuery(t *testing.T) {
	tests := []struct {
		query    string
		insecure bool
	}{
		{"no_verify=1", true},
		{"no_verify=0", false},
		{"foo=no_verify", false},
		{"not_no_verify=1", false},
		{"cacert=/opt/ca-bundle.crt&no_verify=1", true},
	}
	for _, tt := range tests {
		if got := InsecureFromQuery(tt.query); got != tt.insecure {
			t.Errorf("InsecureFromQuery(%q) = %v, want %v", tt.query, got, tt.insecure)
		}
	}
}

func TestParseLibvirtTLS(t *testing.T) {
	insecure, caBundle := ParseLibvirtTLS("vpx://user@vcenter/dc/host?cacert=/opt/ca-bundle.crt")
	if insecure {
		t.Fatal("expected secure mode")
	}
	if caBundle != "" {
		t.Fatalf("caBundle = %q", caBundle)
	}
	insecure, _ = ParseLibvirtTLS("vpx://user@vcenter/dc/host?no_verify=1")
	if !insecure {
		t.Fatal("expected insecure")
	}
}

func TestParseQueryTLS(t *testing.T) {
	insecure, caBundle := ParseQueryTLS("no_verify=1&cacert=/ignored")
	if !insecure || caBundle != "" {
		t.Fatalf("insecure=%v caBundle=%q", insecure, caBundle)
	}
}
