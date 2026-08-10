package driversource

import (
	"os"
	"path/filepath"
	"strings"
)

func sanitizePackageRel(rel string) string {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return ""
	}
	rel = strings.ReplaceAll(rel, `\`, "/")
	rel = filepath.Clean(filepath.FromSlash(rel))
	if rel == "." || rel == ".." {
		return ""
	}
	if filepath.IsAbs(rel) {
		return ""
	}
	// Reject escape from package root (".." path elements after Clean).
	for _, p := range strings.Split(filepath.ToSlash(rel), "/") {
		if p == ".." {
			return ""
		}
	}
	return rel
}

func resolveUnderRoot(root, rel string) (string, bool) {
	rel = sanitizePackageRel(rel)
	if rel == "" || root == "" {
		return "", false
	}
	full := filepath.Join(root, rel)
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	absFull, err := filepath.Abs(full)
	if err != nil {
		return "", false
	}
	sep := string(os.PathSeparator)
	if absFull != absRoot && !strings.HasPrefix(absFull, absRoot+sep) {
		return "", false
	}
	return absFull, true
}
