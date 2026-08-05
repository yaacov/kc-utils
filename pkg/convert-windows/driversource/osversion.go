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

// normalizeOSProductName cleans registry product names for substring matching.
// Windows often reports names like "Windows Server (R) 2008 Enterprise" with a
// trailing NUL from the hive; "(R)" also breaks naive "server 2008" checks.
func normalizeOSProductName(requested string) string {
	s := strings.ReplaceAll(requested, "\x00", "")
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "(r)", "")
	s = strings.ReplaceAll(s, "(tm)", "")
	return strings.Join(strings.Fields(s), " ")
}

// CanonicalOSVersions expands an OS version token or product name into aliases
// used by the virtio-win ISO layout (w10, 2k19, …).
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
		add("2k25", "w11", "10.0")
	case strings.Contains(normalized, "server 2022"):
		add("2k22", "w11", "10.0")
	case strings.Contains(normalized, "server 2019"):
		add("2k19", "w10", "10.0")
	case strings.Contains(normalized, "server 2016"):
		add("2k16", "w10", "10.0")
	case strings.Contains(normalized, "server 2012 r2"):
		add("2k12r2", "w8.1", "6.3")
	case strings.Contains(normalized, "server 2012"):
		add("2k12", "w8", "6.2")
	case strings.Contains(normalized, "server 2008 r2"):
		add("2k8r2", "w7", "6.1")
	case strings.Contains(normalized, "server 2008"):
		add("2k8", "vista", "6.0")
	case strings.Contains(normalized, "windows 11"):
		add("w11", "10.0")
	case strings.Contains(normalized, "windows 10"):
		add("w10", "10.0")
	case strings.Contains(normalized, "windows 8.1"):
		add("w8.1", "6.3")
	case strings.Contains(normalized, "windows 8"):
		add("w8", "6.2")
	case strings.Contains(normalized, "windows 7"):
		add("w7", "6.1")
	case strings.Contains(normalized, "vista"):
		add("vista", "6.0")
	case strings.Contains(normalized, "xp"):
		add("xp", "5.1")
	case normalized == "10.0":
		add("w10", "w11", "2k16", "2k19", "2k22", "2k25")
	case normalized == "2k25":
		add("w11", "10.0")
	case normalized == "2k22":
		add("w11", "10.0")
	case normalized == "2k19":
		add("w10", "10.0")
	case normalized == "2k16":
		add("w10", "10.0")
	case normalized == "2k12r2":
		add("w8.1", "6.3")
	case normalized == "2k12":
		add("w8", "6.2")
	case normalized == "2k8r2":
		add("w7", "6.1")
	case normalized == "2k8":
		add("vista", "6.0")
	case normalized == "w11":
		// bare ISO token — do not attach 10.0 (would match Server 2019)
	case normalized == "w10":
		// bare ISO token
	case normalized == "w8.1":
		add("6.3")
	case normalized == "w8":
		add("6.2")
	case normalized == "w7":
		add("6.1")
	case normalized == "6.3":
		add("w8.1", "2k12r2")
	case normalized == "6.2":
		add("w8", "2k12")
	case normalized == "6.1":
		add("w7", "2k8r2")
	case normalized == "6.0":
		add("vista", "2k8")
	case normalized == "5.1":
		add("xp")
	}

	return aliases
}
