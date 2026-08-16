//go:build unix

package bls

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/configedit/bls"
	"github.com/yaacov/kc-utils/pkg/convert-linux/bootloader"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type BLSHandler struct{}

func init() {
	bootloader.Handlers.Register("bls", &BLSHandler{})
}

func (b *BLSHandler) Detect(guestRoot string) bool {
	entriesDir := filepath.Join(guestRoot, "boot", "loader", "entries")
	entries, err := guest.FileReadDir(entriesDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name, ".conf") {
			return true
		}
	}
	return false
}

func (b *BLSHandler) GetDefaultKernel(guestRoot string) (string, error) {
	entriesDir := filepath.Join(guestRoot, "boot", "loader", "entries")
	entries, err := guest.FileReadDir(entriesDir)
	if err != nil {
		return "", err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	for _, e := range entries {
		if strings.HasSuffix(e.Name, ".conf") {
			content, err := guest.FileRead(filepath.Join(entriesDir, e.Name))
			if err != nil {
				continue
			}
			entry := bls.Parse(string(content))
			if linux := entry.Get("linux"); linux != "" {
				return linux, nil
			}
		}
	}
	return "", nil
}

func (b *BLSHandler) SetDefaultKernel(guestRoot, kernelVersion string) error {
	entriesDir := filepath.Join(guestRoot, "boot", "loader", "entries")
	entries, err := guest.FileReadDir(entriesDir)
	if err != nil {
		return err
	}
	// Prefer matching entry by making it sort first (prefix "0-").
	for _, e := range entries {
		if !strings.HasSuffix(e.Name, ".conf") {
			continue
		}
		path := filepath.Join(entriesDir, e.Name)
		content, err := guest.FileRead(path)
		if err != nil {
			continue
		}
		if !strings.Contains(string(content), kernelVersion) {
			continue
		}
		if strings.HasPrefix(e.Name, "0-") {
			return nil
		}
		newName := "0-" + e.Name
		return guest.FileRename(path, filepath.Join(entriesDir, newName))
	}
	return nil
}

func (b *BLSHandler) AddKernelArg(guestRoot, arg string) error {
	return b.modifyEntries(guestRoot, func(entry *bls.Entry) {
		opts := entry.Get("options")
		if !strings.Contains(opts, arg) {
			entry.Set("options", strings.TrimSpace(opts+" "+arg))
		}
	})
}

func (b *BLSHandler) RemoveKernelArg(guestRoot, prefix string) error {
	return b.modifyEntries(guestRoot, func(entry *bls.Entry) {
		opts := entry.Get("options")
		var filtered []string
		for _, arg := range strings.Fields(opts) {
			if arg != prefix && !strings.HasPrefix(arg, prefix+"=") {
				filtered = append(filtered, arg)
			}
		}
		entry.Set("options", strings.Join(filtered, " "))
	})
}

func (b *BLSHandler) RegenerateConfig(guestRoot string) error {
	// BLS entries are the config; nothing to regenerate.
	return nil
}

func (b *BLSHandler) modifyEntries(guestRoot string, fn func(*bls.Entry)) error {
	entriesDir := filepath.Join(guestRoot, "boot", "loader", "entries")
	entries, err := guest.FileReadDir(entriesDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name, ".conf") {
			continue
		}
		path := filepath.Join(entriesDir, e.Name)
		content, err := guest.FileRead(path)
		if err != nil {
			continue
		}
		entry := bls.Parse(string(content))
		fn(entry)
		if err := guest.FileWrite(path, []byte(entry.String()), 0o644); err != nil {
			return err
		}
	}
	return nil
}
