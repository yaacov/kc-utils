package keyvalue

import (
	"strings"
)

// File represents a key=value config file (e.g., /etc/sysconfig/*).
type File struct {
	lines []string
	sep   string
}

// Parse parses a key=value file. Separator defaults to "=".
func Parse(content string) *File {
	return &File{
		lines: strings.Split(content, "\n"),
		sep:   "=",
	}
}

// Get returns the value for a key, unquoted.
func (f *File) Get(key string) string {
	prefix := key + f.sep
	for _, line := range f.lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			val := trimmed[len(prefix):]
			return strings.Trim(val, "\"'")
		}
	}
	return ""
}

// Set sets or adds a key-value pair.
func (f *File) Set(key, value string) {
	prefix := key + f.sep
	for i, line := range f.lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			f.lines[i] = key + f.sep + "\"" + value + "\""
			return
		}
	}
	f.lines = append(f.lines, key+f.sep+"\""+value+"\"")
}

// Delete removes a key.
func (f *File) Delete(key string) {
	prefix := key + f.sep
	var result []string
	for _, line := range f.lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, prefix) {
			result = append(result, line)
		}
	}
	f.lines = result
}

// String serializes back.
func (f *File) String() string {
	return strings.Join(f.lines, "\n")
}
