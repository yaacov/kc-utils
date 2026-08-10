package driversource

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// parseINFRequirements reads CatalogFile[.nt*] and SourceDisksFiles[.arch]
// entries relevant to arch. Companions are package-relative paths (optional
// SourceDisksFiles subdirectory + filename).
func parseINFRequirements(infPath, arch string) (catalogs, companions []string, err error) {
	f, err := os.Open(infPath)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	inSourceDisks := false
	catSeen := make(map[string]struct{})
	compSeen := make(map[string]struct{})
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			sec := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			inSourceDisks = sourceDisksSectionMatches(sec, arch)
			continue
		}

		if key, val, ok := splitINFKeyValue(line); ok && catalogKeyMatches(key, arch) {
			cat := sanitizePackageRel(filepath.Base(strings.TrimSpace(val)))
			if cat != "" {
				lower := strings.ToLower(cat)
				if _, dup := catSeen[lower]; !dup {
					catSeen[lower] = struct{}{}
					catalogs = append(catalogs, cat)
				}
			}
		}

		if !inSourceDisks {
			continue
		}
		rel := parseSourceDisksEntry(line)
		if rel == "" {
			continue
		}
		lower := strings.ToLower(filepath.ToSlash(rel))
		if _, dup := compSeen[lower]; dup {
			continue
		}
		compSeen[lower] = struct{}{}
		companions = append(companions, rel)
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	return catalogs, companions, nil
}

func sourceDisksSectionMatches(sec, arch string) bool {
	sec = strings.ToLower(strings.TrimSpace(sec))
	if sec == "sourcedisksfiles" {
		return true
	}
	const prefix = "sourcedisksfiles."
	if !strings.HasPrefix(sec, prefix) {
		return false
	}
	suf := sec[len(prefix):]
	switch suf {
	case "x86", "ia64", "amd64", "arm", "arm64":
		// ok
	default:
		return false
	}
	want := sourceDisksArchSuffix(arch)
	if want == "" {
		// Unknown guest arch: accept every recognized decoration.
		return true
	}
	return suf == want
}

func catalogKeyMatches(key, arch string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "catalogfile" {
		return true
	}
	const prefix = "catalogfile."
	if !strings.HasPrefix(key, prefix) {
		return false
	}
	suf := key[len(prefix):]
	switch suf {
	case "nt", "ntx86", "ntia64", "ntamd64", "ntarm", "ntarm64":
		// ok
	default:
		return false
	}
	want := catalogArchSuffix(arch)
	if want == "" {
		return true
	}
	return suf == "nt" || suf == want
}

func sourceDisksArchSuffix(arch string) string {
	switch NormalizeArch(arch) {
	case "amd64":
		return "amd64"
	case "x86":
		return "x86"
	case "arm64":
		return "arm64"
	case "arm":
		return "arm"
	default:
		return ""
	}
}

func catalogArchSuffix(arch string) string {
	switch NormalizeArch(arch) {
	case "amd64":
		return "ntamd64"
	case "x86":
		return "ntx86"
	case "arm64":
		return "ntarm64"
	case "arm":
		return "ntarm"
	default:
		return ""
	}
}

func splitINFKeyValue(line string) (key, val string, ok bool) {
	idx := strings.IndexByte(line, '=')
	if idx < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	val = strings.TrimSpace(line[idx+1:])
	// Strip trailing comments.
	if c := strings.IndexByte(val, ';'); c >= 0 {
		val = strings.TrimSpace(val[:c])
	}
	if key == "" || val == "" {
		return "", "", false
	}
	return key, val, true
}

// parseSourceDisksEntry parses:
//
//	filename = diskid[,[subdir][,size]]
//
// and returns a package-relative path (subdir/filename when subdir is set).
func parseSourceDisksEntry(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, ";") {
		return ""
	}
	if c := strings.IndexByte(line, ';'); c >= 0 {
		line = strings.TrimSpace(line[:c])
	}

	var left, right string
	if eq := strings.IndexByte(line, '='); eq >= 0 {
		left = strings.TrimSpace(line[:eq])
		right = strings.TrimSpace(line[eq+1:])
	} else {
		// Bare "foo.sys,,,2" form: filename before first comma.
		left = line
		if c := strings.IndexByte(left, ','); c >= 0 {
			left = strings.TrimSpace(left[:c])
		}
	}
	if left == "" {
		return ""
	}
	// Filename token ends at whitespace/comma if no '=' was present above.
	if end := strings.IndexAny(left, " \t,"); end >= 0 {
		left = strings.TrimSpace(left[:end])
	}
	filename := filepath.Base(left)
	if filename == "" || filename == "." || !strings.Contains(filename, ".") {
		return ""
	}

	subdir := ""
	if right != "" {
		parts := strings.Split(right, ",")
		if len(parts) >= 2 {
			subdir = strings.TrimSpace(parts[1])
		}
	}
	if subdir == "" {
		return sanitizePackageRel(filename)
	}
	// INF paths use Windows separators; normalize before joining on the host.
	subdir = strings.ReplaceAll(subdir, `\`, "/")
	return sanitizePackageRel(filepath.Join(filepath.FromSlash(subdir), filename))
}
