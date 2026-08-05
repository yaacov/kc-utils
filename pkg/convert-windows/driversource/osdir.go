package driversource

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FindBestOSDir picks the best virtio-win by-os/<arch>/<osVersion>/ directory for a guest.
func FindBestOSDir(archDir, osVersion string) (string, error) {
	entries, err := os.ReadDir(archDir)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", archDir, err)
	}

	var matches []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if MatchOSVersion(entry.Name(), osVersion) {
			matches = append(matches, entry.Name())
		}
	}
	if len(matches) > 0 {
		sort.Strings(matches)
		return filepath.Join(archDir, matches[0]), nil
	}

	for _, fallback := range osDirFallbacks(osVersion) {
		dir := filepath.Join(archDir, fallback)
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			return dir, nil
		}
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return "", fmt.Errorf("no virtio-win OS directory for %q under %s (available: %s)",
		osVersion, archDir, strings.Join(names, ", "))
}

func osDirFallbacks(osVersion string) []string {
	for _, alias := range CanonicalOSVersions(osVersion) {
		if alias == "2k8" {
			return []string{"2k8R2"}
		}
	}
	return nil
}
