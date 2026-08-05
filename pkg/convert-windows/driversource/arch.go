package driversource

import "strings"

// NormalizeArch maps guest inspection arch names to virtio-win directory names
// and vice versa (e.g. x86_64 ↔ amd64, i386 ↔ x86).
func NormalizeArch(arch string) string {
	switch strings.ToLower(strings.TrimSpace(arch)) {
	case "x86_64", "amd64", "x64":
		return "amd64"
	case "i386", "i486", "i586", "i686", "x86":
		return "x86"
	case "aarch64", "arm64":
		return "arm64"
	default:
		return strings.ToLower(strings.TrimSpace(arch))
	}
}

// ArchMatches reports whether a virtio-win directory arch matches a guest arch.
func ArchMatches(dirArch, guestArch string) bool {
	return NormalizeArch(dirArch) == NormalizeArch(guestArch)
}

// ArchSearchNames returns directory names to try for a guest arch.
func ArchSearchNames(guestArch string) []string {
	norm := NormalizeArch(guestArch)
	names := []string{norm}
	switch norm {
	case "amd64":
		names = append(names, "x86_64")
	case "x86":
		names = append(names, "i386")
	}
	raw := strings.ToLower(strings.TrimSpace(guestArch))
	if raw != "" && raw != norm {
		names = append(names, raw)
	}
	return names
}
