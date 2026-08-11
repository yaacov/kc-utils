package single

import (
	"fmt"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/prepare/root"
)

type Selector struct{}

func init() {
	root.Selectors.Register("single", &Selector{})
}

func (s *Selector) Select(candidates []types.RootCandidate, _ string) (types.RootCandidate, error) {
	switch len(candidates) {
	case 0:
		return types.RootCandidate{}, fmt.Errorf("no root candidates found")
	case 1:
		return candidates[0], nil
	default:
		return types.RootCandidate{}, &root.MultiBootError{Candidates: candidates}
	}
}
