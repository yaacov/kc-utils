//go:build unix

package inspect

import (
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/guest/guestio"
)

// DetectArch determines the guest architecture from the mounted filesystem.
func DetectArch(guestRoot string) string {
	for _, path := range []string{
		filepath.Join(guestRoot, "lib", "modules"),
		filepath.Join(guestRoot, "usr", "lib", "modules"),
	} {
		entries, err := guestio.FileReadDir(path)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name
			switch {
			case strings.Contains(name, "x86_64") || strings.Contains(name, "amd64"):
				return "x86_64"
			case strings.Contains(name, "aarch64") || strings.Contains(name, "arm64"):
				return "aarch64"
			case strings.Contains(name, "ppc64le"):
				return "ppc64le"
			case strings.Contains(name, "s390x"):
				return "s390x"
			}
		}
	}
	return "unknown"
}
