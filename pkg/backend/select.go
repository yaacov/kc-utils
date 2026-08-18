//go:build unix

package backend

import (
	"fmt"
	"strings"
)

// ValidateName reports whether name is a known backend identifier.
func ValidateName(name string) error {
	switch strings.TrimSpace(name) {
	case NameDirect, NameGuestfs:
		return nil
	default:
		return fmt.Errorf("unknown backend %q (want %q or %q)", name, NameDirect, NameGuestfs)
	}
}

// Lookup returns a registered backend plugin without checking runtime availability.
func Lookup(name string) (Plugin, error) {
	name = strings.TrimSpace(name)
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	p, ok := Plugins.Get(name)
	if !ok {
		return nil, fmt.Errorf("backend %q not registered (missing plugin import?)", name)
	}
	return p, nil
}

// Resolve returns a registered backend plugin after validating name and runtime availability.
func Resolve(name string) (Plugin, error) {
	p, err := Lookup(name)
	if err != nil {
		return nil, err
	}
	if !p.Available() {
		_, reason := checkRequirements(p.Requirements())
		if reason == "" {
			reason = "runtime requirements not met"
		}
		return nil, fmt.Errorf("backend %s: %s", name, reason)
	}
	return p, nil
}
