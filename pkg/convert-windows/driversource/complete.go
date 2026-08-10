package driversource

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FilterComplete keeps only packages whose required offline files all exist.
// Incomplete packages are dropped with a warning. Kept packages have Files set
// to the host paths that must be staged into the guest.
func FilterComplete(files []DriverFile) []DriverFile {
	if len(files) == 0 {
		return nil
	}
	kept := make([]DriverFile, 0, len(files))
	for i := range files {
		df := &files[i]
		required, missing := resolveRequiredFiles(df)
		if len(missing) > 0 {
			slog.Warn("skipping incomplete virtio package",
				"name", df.Name,
				"inf", df.InfPath,
				"missing", missing)
			continue
		}
		if len(required) == 0 {
			slog.Warn("skipping virtio package with no installable files",
				"name", df.Name,
				"inf", df.InfPath)
			continue
		}
		df.Files = required
		kept = append(kept, *df)
	}
	return kept
}

func resolveRequiredFiles(df *DriverFile) (required, missing []string) {
	if df.InfPath == "" {
		return nil, []string{"(empty InfPath)"}
	}
	if isMSIPackage(df) {
		if _, err := os.Stat(df.InfPath); err != nil {
			return nil, []string{df.InfPath}
		}
		return []string{df.InfPath}, nil
	}

	baseDir := df.SrcPath
	if baseDir == "" {
		baseDir = filepath.Dir(df.InfPath)
	}

	rels := []string{filepath.Base(df.InfPath)}
	seen := map[string]struct{}{strings.ToLower(rels[0]): {}}
	catalogs, companions, err := parseINFRequirements(df.InfPath, df.Arch)
	if err != nil {
		return nil, []string{df.InfPath + ": " + err.Error()}
	}
	addRel := func(rel string) {
		rel = sanitizePackageRel(rel)
		if rel == "" {
			return
		}
		lower := strings.ToLower(filepath.ToSlash(rel))
		if _, ok := seen[lower]; ok {
			return
		}
		seen[lower] = struct{}{}
		rels = append(rels, rel)
	}
	for _, name := range catalogs {
		addRel(name)
	}
	for _, name := range companions {
		addRel(name)
	}

	for _, rel := range rels {
		path, ok := resolveUnderRoot(baseDir, rel)
		if !ok {
			missing = append(missing, filepath.Join(baseDir, rel))
			continue
		}
		if _, err := os.Stat(path); err != nil {
			missing = append(missing, path)
			continue
		}
		required = append(required, path)
	}
	sort.Strings(missing)
	return required, missing
}

func isMSIPackage(df *DriverFile) bool {
	if df.Name == "qemu-ga" {
		return true
	}
	return strings.HasSuffix(strings.ToLower(df.InfPath), ".msi")
}
