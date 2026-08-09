package copy

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/yaacov/kc-utils/pkg/v2v/config"
)

func resolveCaBundle(caBundle string) string {
	if caBundle == "" {
		return config.DefaultCaBundle
	}
	return caBundle
}

func tlsConfig(insecure bool, caBundle string) (*tls.Config, error) {
	if insecure {
		return &tls.Config{InsecureSkipVerify: true}, nil
	}
	path := resolveCaBundle(caBundle)
	pem, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read CA bundle %s: %w", path, err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("parse CA bundle %s: no valid PEM certificates", path)
	}
	return &tls.Config{RootCAs: pool}, nil
}

func newHTTPClient(insecure bool, caBundle string) (*http.Client, error) {
	cfg, err := tlsConfig(insecure, caBundle)
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: cfg,
		},
	}, nil
}
