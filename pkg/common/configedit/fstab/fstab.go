package fstab

import (
	"fmt"
	"strings"
)

// Entry represents a single line in /etc/fstab.
type Entry struct {
	Device     string
	MountPoint string
	FSType     string
	Options    string
	Dump       string
	Pass       string
	Comment    string // non-empty if this line is a comment
}

// File represents a parsed /etc/fstab.
type File struct {
	Entries []Entry
}

// Parse parses fstab content.
func Parse(content string) *File {
	f := &File{}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			f.Entries = append(f.Entries, Entry{Comment: ""})
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			f.Entries = append(f.Entries, Entry{Comment: line})
			continue
		}
		fields := strings.Fields(trimmed)
		e := Entry{}
		if len(fields) >= 1 {
			e.Device = fields[0]
		}
		if len(fields) >= 2 {
			e.MountPoint = fields[1]
		}
		if len(fields) >= 3 {
			e.FSType = fields[2]
		}
		if len(fields) >= 4 {
			e.Options = fields[3]
		}
		if len(fields) >= 5 {
			e.Dump = fields[4]
		}
		if len(fields) >= 6 {
			e.Pass = fields[5]
		}
		f.Entries = append(f.Entries, e)
	}
	return f
}

// RemapDevice replaces device path prefixes in the Device field (column 1).
func (f *File) RemapDevice(oldPrefix, newPrefix string) {
	for i := range f.Entries {
		if f.Entries[i].Comment != "" || f.Entries[i].Device == "" {
			continue
		}
		if strings.HasPrefix(f.Entries[i].Device, oldPrefix) {
			f.Entries[i].Device = newPrefix + f.Entries[i].Device[len(oldPrefix):]
		}
	}
}

// RemapAllFields replaces device path prefixes in all text fields of each
// entry. This is useful for files like /etc/crypttab where device paths can
// appear in any column.
func (f *File) RemapAllFields(oldPrefix, newPrefix string) {
	for i := range f.Entries {
		if f.Entries[i].Comment != "" || f.Entries[i].Device == "" {
			continue
		}
		f.Entries[i].Device = remapField(f.Entries[i].Device, oldPrefix, newPrefix)
		f.Entries[i].MountPoint = remapField(f.Entries[i].MountPoint, oldPrefix, newPrefix)
		f.Entries[i].FSType = remapField(f.Entries[i].FSType, oldPrefix, newPrefix)
		f.Entries[i].Options = remapField(f.Entries[i].Options, oldPrefix, newPrefix)
	}
}

func remapField(field, oldPrefix, newPrefix string) string {
	if strings.HasPrefix(field, oldPrefix) {
		return newPrefix + field[len(oldPrefix):]
	}
	return field
}

// DeviceEntries returns non-comment, non-empty entries.
func (f *File) DeviceEntries() []Entry {
	var result []Entry
	for _, e := range f.Entries {
		if e.Device != "" && e.Comment == "" {
			result = append(result, e)
		}
	}
	return result
}

// String serializes back to fstab format.
func (f *File) String() string {
	var lines []string
	for _, e := range f.Entries {
		if e.Comment != "" {
			lines = append(lines, e.Comment)
			continue
		}
		if e.Device == "" {
			lines = append(lines, "")
			continue
		}
		dump := e.Dump
		if dump == "" {
			dump = "0"
		}
		pass := e.Pass
		if pass == "" {
			pass = "0"
		}
		opts := e.Options
		if opts == "" {
			opts = "defaults"
		}
		lines = append(lines, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s",
			e.Device, e.MountPoint, e.FSType, opts, dump, pass))
	}
	return strings.Join(lines, "\n") + "\n"
}
