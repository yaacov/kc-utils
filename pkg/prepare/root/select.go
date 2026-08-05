package root

import (
	"fmt"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/plugin"
	"github.com/yaacov/kc-utils/pkg/common/types"
)

// Selectors is the global registry of RootSelector implementations.
var Selectors = plugin.NewRegistry[string, RootSelector]()

// RootSelector applies a root selection policy to discovered candidates.
type RootSelector interface {
	Select(candidates []types.RootCandidate, choice string) (types.RootCandidate, error)
}

// Select resolves options.root and picks a root candidate.
func Select(candidates []types.RootCandidate, choice string) (types.RootCandidate, error) {
	choice = strings.TrimSpace(choice)
	if choice == "" {
		choice = "first"
	}

	switch {
	case choice == "single":
		sel, ok := Selectors.Get("single")
		if !ok {
			return types.RootCandidate{}, fmt.Errorf("single root selector not registered")
		}
		return sel.Select(candidates, choice)
	case choice == "first":
		sel, ok := Selectors.Get("first")
		if !ok {
			return types.RootCandidate{}, fmt.Errorf("first root selector not registered")
		}
		return sel.Select(candidates, choice)
	case strings.HasPrefix(choice, "/dev/"):
		sel, ok := Selectors.Get("device")
		if !ok {
			return types.RootCandidate{}, fmt.Errorf("device root selector not registered")
		}
		return sel.Select(candidates, choice)
	default:
		return types.RootCandidate{}, formatInvalidChoice(choice, candidates)
	}
}

func formatInvalidChoice(choice string, candidates []types.RootCandidate) error {
	return fmt.Errorf("invalid options.root %q: use \"single\", \"first\", or a device path (%s)",
		choice, formatCandidates(candidates))
}

func formatCandidates(candidates []types.RootCandidate) string {
	if len(candidates) == 0 {
		return "no candidates"
	}
	var parts []string
	for i := range candidates {
		parts = append(parts, fmt.Sprintf("[%d] %s (%s)", i+1, candidates[i].DevicePath, candidates[i].ProductName))
	}
	return strings.Join(parts, "; ")
}

// MultiBootError is returned when single policy finds multiple roots.
type MultiBootError struct {
	Candidates []types.RootCandidate
}

func (e *MultiBootError) Error() string {
	return fmt.Sprintf("multiple operating systems found; set options.root to \"first\" or a device path (%s)",
		formatCandidates(e.Candidates))
}
