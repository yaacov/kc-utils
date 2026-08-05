package decryptor

import "github.com/yaacov/kc-utils/pkg/common/plugin"

type Decryptor interface {
	Decrypt(device, keySource string) (string, error)
	Close() error
}

var Decryptors = plugin.NewRegistry[string, Decryptor]()
