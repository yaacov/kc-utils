package env

import (
	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/prepare/guest/bitlocker"
)

// BuildBitLockerSpec builds BitLocker configuration from the key directory.
func BuildBitLockerSpec(cfg *Config) (*types.BitLockerSpec, error) {
	m, err := bitlocker.KeyFilesMap(cfg.BitLockerDir)
	if err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return nil, nil
	}
	return &types.BitLockerSpec{KeyFiles: m}, nil
}
