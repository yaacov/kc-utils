package device

import (
	"fmt"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/prepare/root"
)

type Selector struct{}

func init() {
	root.Selectors.Register("device", &Selector{})
}

func (s *Selector) Select(candidates []types.RootCandidate, choice string) (types.RootCandidate, error) {
	for i := range candidates {
		if candidates[i].DevicePath == choice {
			return candidates[i], nil
		}
	}
	var parts []string
	for i := range candidates {
		parts = append(parts, fmt.Sprintf("[%d] %s (%s)", i+1, candidates[i].DevicePath, candidates[i].ProductName))
	}
	return types.RootCandidate{}, fmt.Errorf("root device %q not found among candidates (%s)",
		choice, strings.Join(parts, "; "))
}
