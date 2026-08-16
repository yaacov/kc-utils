package version

import (
	"strings"
)

// NormalizeProductName cleans registry product names for substring matching.
func NormalizeProductName(productName string) string {
	s := strings.ReplaceAll(productName, "\x00", "")
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "(r)", "")
	s = strings.ReplaceAll(s, "(tm)", "")
	return strings.Join(strings.Fields(s), " ")
}

func isServerProduct(productName string) bool {
	n := NormalizeProductName(productName)
	return strings.Contains(n, "server")
}

// usesWin11DriverSet reports whether a product should receive the Windows 11
// driver set, which also covers Server 2022/2025 (same driver generation).
func usesWin11DriverSet(productName string) bool {
	n := NormalizeProductName(productName)
	return strings.Contains(n, "windows 11") || strings.Contains(n, "server 2022") ||
		strings.Contains(n, "server 2025")
}
