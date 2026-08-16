//go:build unix

package guest

import (
	"path/filepath"
	"strings"
)

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
