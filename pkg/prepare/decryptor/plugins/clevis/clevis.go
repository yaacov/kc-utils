//go:build linux

package clevis

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/guest"
	"github.com/yaacov/kc-utils/pkg/prepare/decryptor"
)

type ClevisDecryptor struct{}

func init() {
	decryptor.Decryptors.Register("clevis", &ClevisDecryptor{})
}

func (c *ClevisDecryptor) Decrypt(device, _ string) (string, error) {
	g := guest.Active()
	if g == nil {
		return "", fmt.Errorf("no active guest handle")
	}
	return g.UnlockClevis(device, mapperName(device))
}

func (c *ClevisDecryptor) Close() error { return nil }

func mapperName(device string) string {
	return "v2v-luks-clevis-" + sanitizeMapper(device) + "-" + deviceSuffix(device)
}

func deviceSuffix(device string) string {
	sum := sha256.Sum256([]byte(device))
	return hex.EncodeToString(sum[:4])
}

func sanitizeMapper(device string) string {
	base := filepath.Base(device)
	base = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, base)
	if base == "" {
		return "vol"
	}
	return base
}
