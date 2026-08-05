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

func isWindows11Product(productName string) bool {
	n := NormalizeProductName(productName)
	return strings.Contains(n, "windows 11") || strings.Contains(n, "server 2022") ||
		strings.Contains(n, "server 2025")
}
