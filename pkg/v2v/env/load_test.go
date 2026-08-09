package env

import (
	"os"
	"testing"
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
