package copy

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

func TestTLSConfigInsecure(t *testing.T) {
	cfg, err := tlsConfig(true, "")
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.InsecureSkipVerify {
		t.Fatal("expected InsecureSkipVerify=true")
	}
}

func TestTLSConfigValidBundle(t *testing.T) {
	bundle := writeTestCABundle(t, t.TempDir())

	cfg, err := tlsConfig(false, bundle)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.InsecureSkipVerify {
		t.Fatal("expected verification enabled")
	}
	if cfg.RootCAs == nil {
		t.Fatal("expected RootCAs pool")
	}
}

func TestTLSConfigMissingBundle(t *testing.T) {
	_, err := tlsConfig(false, filepath.Join(t.TempDir(), "missing.crt"))
	if err == nil {
		t.Fatal("expected error for missing CA bundle")
	}
}

func TestTLSConfigInvalidPEM(t *testing.T) {
	dir := t.TempDir()
	bundle := filepath.Join(dir, "ca-bundle.crt")
	if err := os.WriteFile(bundle, []byte("not a pem file"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := tlsConfig(false, bundle)
	if err == nil {
		t.Fatal("expected error for invalid PEM")
	}
}

func TestNewHTTPClientInsecure(t *testing.T) {
	client, err := newHTTPClient(true, "")
	if err != nil {
		t.Fatal(err)
	}
	if client.Transport == nil {
		t.Fatal("expected transport")
	}
}
