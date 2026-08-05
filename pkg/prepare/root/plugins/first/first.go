package first

import (
	"fmt"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/prepare/root"
)

type Selector struct{}

func init() {
	root.Selectors.Register("first", &Selector{})
}

func (s *Selector) Select(candidates []types.RootCandidate, _ string) (types.RootCandidate, error) {
	if len(candidates) == 0 {
		return types.RootCandidate{}, fmt.Errorf("no root candidates found")
	}
	return candidates[0], nil
}
