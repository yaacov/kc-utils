package bls

import (
	"fmt"
	"strings"
)

// Entry represents a Boot Loader Spec entry file.
type Entry struct {
	fields []field
}

type field struct {
	key   string
	value string
}

// Parse parses a BLS entry file.
func Parse(content string) *Entry {
	e := &Entry{}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.SplitN(trimmed, " ", 2)
		val := ""
		if len(parts) == 2 {
			val = parts[1]
		}
		e.fields = append(e.fields, field{key: parts[0], value: val})
	}
	return e
}

// Get returns the value of a key.
func (e *Entry) Get(key string) string {
	for _, f := range e.fields {
		if f.key == key {
			return f.value
		}
	}
	return ""
}

// Set sets or adds a key-value pair.
func (e *Entry) Set(key, value string) {
	for i, f := range e.fields {
		if f.key == key {
			e.fields[i].value = value
			return
		}
	}
	e.fields = append(e.fields, field{key: key, value: value})
}

// String serializes back to BLS format.
func (e *Entry) String() string {
	var lines []string
	for _, f := range e.fields {
		lines = append(lines, fmt.Sprintf("%s %s", f.key, f.value))
	}
	return strings.Join(lines, "\n") + "\n"
}
