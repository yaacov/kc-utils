//go:build linux

package keyfile

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/guest"
	"github.com/yaacov/kc-utils/pkg/prepare/decryptor"
)

type KeyFileDecryptor struct{}

func init() {
	decryptor.Decryptors.Register("keyfile", &KeyFileDecryptor{})
}

func (k *KeyFileDecryptor) Decrypt(device, keySource string) (string, error) {
	g := guest.Active()
	if g == nil {
		return "", fmt.Errorf("no active guest handle")
	}
	return g.Decrypt(device, keySource, mapperNameFor(device))
}

func (k *KeyFileDecryptor) Close() error { return nil }

func mapperNameFor(device string) string {
	base := filepath.Base(device)
	base = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, base)
	if base == "" {
		base = "vol"
	}
	return "v2v-luks-keyfile-" + base
}
