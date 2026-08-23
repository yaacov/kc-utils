package luks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScanKeyFiles returns absolute paths of key files in luksDir (e.g. /etc/luks).
func ScanKeyFiles(luksDir string) ([]string, error) {
	info, err := os.Stat(luksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat luks dir: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", luksDir)
	}
	entries, err := os.ReadDir(luksDir)
	if err != nil {
		return nil, fmt.Errorf("read luks dir: %w", err)
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), "..") {
			continue
		}
		paths = append(paths, filepath.Join(luksDir, e.Name()))
	}
	return paths, nil
}

// KeyFilesMap builds a device→keyfile map. When device is empty, uses "all" sentinel
// so kc-prepare tries every key on every encrypted volume.
func KeyFilesMap(luksDir string) (map[string]string, error) {
	files, err := ScanKeyFiles(luksDir)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, nil
	}
	m := make(map[string]string, len(files))
	for i, f := range files {
		key := "all"
		if len(files) > 1 {
			key = fmt.Sprintf("all-%d", i)
		}
		m[key] = f
	}
	return m, nil
}
