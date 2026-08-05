//go:build linux

package clevis

import (
	"fmt"

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
	return g.UnlockClevis(device, "v2v-luks-clevis")
}

func (c *ClevisDecryptor) Close() error { return nil }
