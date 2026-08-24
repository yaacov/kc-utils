package bitlocker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScanKeyFiles returns absolute paths of passphrase files in bitlockerDir.
func ScanKeyFiles(bitlockerDir string) ([]string, error) {
	info, err := os.Stat(bitlockerDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat bitlocker dir: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", bitlockerDir)
	}
	entries, err := os.ReadDir(bitlockerDir)
	if err != nil {
		return nil, fmt.Errorf("read bitlocker dir: %w", err)
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), "..") {
			continue
		}
		paths = append(paths, filepath.Join(bitlockerDir, e.Name()))
	}
	return paths, nil
}

// KeyFilesMap builds a device→keyfile map. When multiple keys are present,
// uses "all-N" sentinels so kc-prepare tries each key on every BitLocker volume.
func KeyFilesMap(bitlockerDir string) (map[string]string, error) {
	files, err := ScanKeyFiles(bitlockerDir)
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
