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
	return FindBestOSDirWithPrefs(archDir, osVersion, nil, nil)
}

// FindBestOSDirWithPrefs prefers explicit handler OS dirs before generic alias matching.
func FindBestOSDirWithPrefs(archDir, osVersion string, prefs, fallbacks []string) (string, error) {
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
		if preferred := pickPreferredDir(matches, prefs); preferred != "" {
			return filepath.Join(archDir, preferred), nil
		}
		sort.Strings(matches)
		return filepath.Join(archDir, matches[0]), nil
	}

	for _, dirName := range append(append([]string{}, prefs...), fallbacks...) {
		dir := filepath.Join(archDir, dirName)
		if st, statErr := os.Stat(dir); statErr == nil && st.IsDir() {
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

func pickPreferredDir(matches, prefs []string) string {
	if len(prefs) == 0 {
		return ""
	}
	matchSet := make(map[string]struct{}, len(matches))
	for _, m := range matches {
		matchSet[strings.ToLower(m)] = struct{}{}
	}
	for _, pref := range prefs {
		if _, ok := matchSet[strings.ToLower(pref)]; ok {
			for _, m := range matches {
				if strings.EqualFold(m, pref) {
					return m
				}
			}
		}
	}
	return ""
}
