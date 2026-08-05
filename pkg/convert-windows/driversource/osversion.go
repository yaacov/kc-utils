package driversource

import "strings"

// MatchOSVersion reports whether a virtio-win ISO OS directory matches the guest OS.
func MatchOSVersion(dirVer, requested string) bool {
	dirAliases := CanonicalOSVersions(dirVer)
	reqAliases := CanonicalOSVersions(requested)
	for _, dirAlias := range dirAliases {
		for _, reqAlias := range reqAliases {
			if dirAlias == reqAlias {
				return true
			}
		}
	}
	return false
}

// NormalizeOSProductName cleans registry product names for substring matching.
func NormalizeOSProductName(requested string) string {
	return normalizeOSProductName(requested)
}

// normalizeOSProductName cleans registry product names for substring matching.
func normalizeOSProductName(requested string) string {
	s := strings.ReplaceAll(requested, "\x00", "")
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "(r)", "")
	s = strings.ReplaceAll(s, "(tm)", "")
	return strings.Join(strings.Fields(s), " ")
}

// CanonicalOSVersions maps a guest product name or by-os directory token to the
// virtio-win driver directory name for that version only. There is no shared
// alias bucket (Server 2008 does not match Vista, Win7 does not match 2k8R2, …).
func CanonicalOSVersions(requested string) []string {
	normalized := normalizeOSProductName(requested)
	if normalized == "" {
		return nil
	}

	aliases := []string{normalized}
	add := func(values ...string) {
		for _, value := range values {
			value = strings.ToLower(strings.TrimSpace(value))
			if value == "" {
				continue
			}
			for _, existing := range aliases {
				if existing == value {
					goto nextValue
				}
			}
			aliases = append(aliases, value)
		nextValue:
		}
	}

	switch {
	case strings.Contains(normalized, "server 2025"):
		add("2k25")
	case strings.Contains(normalized, "server 2022"):
		add("2k22")
	case strings.Contains(normalized, "server 2019"):
		add("2k19")
	case strings.Contains(normalized, "server 2016"):
		add("2k16")
	case strings.Contains(normalized, "server 2012 r2"):
		add("2k12r2")
	case strings.Contains(normalized, "server 2012"):
		add("2k12")
	case strings.Contains(normalized, "server 2008 r2"):
		add("2k8r2")
	case strings.Contains(normalized, "server 2008"):
		add("2k8")
	case strings.Contains(normalized, "server 2003"):
		add("2k3")
	case strings.Contains(normalized, "windows 11"):
		add("w11")
	case strings.Contains(normalized, "windows 10"):
		add("w10")
	case strings.Contains(normalized, "windows 8.1"):
		add("w8.1")
	case strings.Contains(normalized, "windows 8"):
		add("w8")
	case strings.Contains(normalized, "windows 7"):
		add("w7")
	case strings.Contains(normalized, "vista"):
		add("vista")
	case strings.Contains(normalized, "xp"):
		add("xp")
	}

	return aliases
}
