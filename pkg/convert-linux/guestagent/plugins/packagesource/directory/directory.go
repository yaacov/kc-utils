package directory

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/yaacov/kc-utils/pkg/convert-linux/guestagent"
)

const DefaultBasePath = "/usr/share/kc-packages"

type DirectorySource struct {
	BasePath string
}

func init() {
	guestagent.Sources.Register("directory", &DirectorySource{})
}

func (d *DirectorySource) basePath() string {
	if d.BasePath != "" {
		return d.BasePath
	}
	if p := strings.TrimSpace(os.Getenv("KC_PACKAGES")); p != "" {
		return p
	}
	return DefaultBasePath
}

func (d *DirectorySource) Available() bool {
	_, err := os.Stat(d.basePath())
	return err == nil
}

func (d *DirectorySource) FindPackages(req guestagent.FindRequest) ([]guestagent.PackageFile, error) {
	// Versioned EL layout (rpm/el{N}/{arch}/) is RPM-only; DEB uses the flat layout below.
	if req.Format == "rpm" {
		if pkg, ok := d.findBestEL(req); ok {
			return []guestagent.PackageFile{pkg}, nil
		}
		slog.Debug("no versioned EL directory matched, falling back to flat layout",
			"format", req.Format, "arch", req.Arch)
	}

	// Flat layout: $base/{format}/{arch}/ and noarch/all.
	var results []guestagent.PackageFile
	for _, searchArch := range archVariants(req.Format, req.Arch) {
		found, err := scanDir(filepath.Join(d.basePath(), req.Format, searchArch), req.Name, req.Format, "")
		if err != nil {
			continue
		}
		results = append(results, found...)
	}
	noarch := noarchName(req.Format)
	found, err := scanDir(filepath.Join(d.basePath(), req.Format, noarch), req.Name, req.Format, "")
	if err == nil {
		results = append(results, found...)
	}
	if len(results) == 0 {
		return nil, nil
	}
	// Flat layout may contain multiple files; pick one for safety.
	return []guestagent.PackageFile{results[0]}, nil
}

func (d *DirectorySource) findBestEL(req guestagent.FindRequest) (guestagent.PackageFile, bool) {
	wantMajor := req.MajorVersion
	if wantMajor <= 0 {
		wantMajor = 0
	}
	supported := d.availableELMajors()
	if len(supported) == 0 {
		return guestagent.PackageFile{}, false
	}
	candidates := elMajorCandidates(wantMajor, supported)
	for _, major := range candidates {
		tag := fmt.Sprintf("el%d", major)
		for _, searchArch := range archVariants(req.Format, req.Arch) {
			dir := filepath.Join(d.basePath(), req.Format, tag, searchArch)
			found, err := scanDir(dir, req.Name, req.Format, tag)
			if err != nil || len(found) == 0 {
				continue
			}
			return found[0], true
		}
		noarchDir := filepath.Join(d.basePath(), req.Format, tag, noarchName(req.Format))
		found, err := scanDir(noarchDir, req.Name, req.Format, tag)
		if err == nil && len(found) > 0 {
			return found[0], true
		}
	}
	return guestagent.PackageFile{}, false
}

// availableELMajors scans $base/rpm/ for el{N} subdirectories at runtime
// instead of relying on a hardcoded list.
func (d *DirectorySource) availableELMajors() []int {
	rpmDir := filepath.Join(d.basePath(), "rpm")
	entries, err := os.ReadDir(rpmDir)
	if err != nil {
		return nil
	}
	var majors []int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "el") {
			continue
		}
		if n, err := strconv.Atoi(name[2:]); err == nil && n > 0 {
			majors = append(majors, n)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(majors)))
	return majors
}

// elMajorCandidates returns preferred EL majors: exact match first, then nearest lower.
func elMajorCandidates(wantMajor int, supported []int) []int {
	if wantMajor > 0 {
		var out []int
		if containsInt(supported, wantMajor) {
			out = append(out, wantMajor)
		}
		var lower []int
		for _, m := range supported {
			if m < wantMajor {
				lower = append(lower, m)
			}
		}
		sort.Sort(sort.Reverse(sort.IntSlice(lower)))
		out = append(out, lower...)
		return out
	}
	// Unknown major: try newest supported first.
	out := append([]int(nil), supported...)
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	return out
}

func containsInt(list []int, v int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func scanDir(dir, name, format, elTag string) ([]guestagent.PackageFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var results []guestagent.PackageFile
	ext := "." + format
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fileName := entry.Name()
		if !strings.HasSuffix(fileName, ext) {
			continue
		}
		if !strings.Contains(fileName, name) {
			continue
		}
		// Arch is inferred from the directory name (e.g. "x86_64", "amd64", "noarch").
		results = append(results, guestagent.PackageFile{
			Name:     name,
			FileName: fileName,
			HostPath: filepath.Join(dir, fileName),
			Format:   format,
			Arch:     filepath.Base(dir),
			ELTag:    elTag,
		})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].FileName < results[j].FileName
	})
	return results, nil
}

func archVariants(format, arch string) []string {
	if format == "deb" {
		switch arch {
		case "x86_64":
			return []string{"amd64"}
		case "aarch64":
			return []string{"arm64"}
		default:
			return []string{arch}
		}
	}
	return []string{arch}
}

func noarchName(format string) string {
	if format == "deb" {
		return "all"
	}
	return "noarch"
}
