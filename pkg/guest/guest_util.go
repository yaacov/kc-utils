//go:build unix

package guest

import (
	"path/filepath"
	"strings"
)

// NormalizeGuestPath cleans p into an absolute slash-separated guest path
// ("/" for empty or "."). Exported for the guestio helpers.
func NormalizeGuestPath(p string) string {
	return normalizeGuestPath(p)
}

func normalizeGuestPath(p string) string {
	p = filepath.ToSlash(p)
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	cleaned := filepath.ToSlash(filepath.Clean(p))
	if cleaned == "." {
		return "/"
	}
	return cleaned
}
