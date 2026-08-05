package env

import (
	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/prepare/guest/luks"
)

// BuildLUKSSpec builds LUKS configuration from env and key directory.
func BuildLUKSSpec(cfg *Config) (*types.LUKSSpec, error) {
	if cfg.NbdeClevis {
		return &types.LUKSSpec{Clevis: true}, nil
	}
	m, err := luks.KeyFilesMap(cfg.LuksDir)
	if err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return nil, nil
	}
	return &types.LUKSSpec{KeyFiles: m}, nil
}
