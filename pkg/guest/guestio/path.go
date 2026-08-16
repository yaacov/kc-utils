//go:build unix

package guestio

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yaacov/kc-utils/pkg/guest"
)

// guestPathFromHost translates a host-style path under the active guest RootPath
// into a guest-absolute path. ok is false when there is no active guest or path
// is outside the guest root (caller should use host os.*).
func guestPathFromHost(path string) (guestPath string, g *guest.Guest, ok bool) {
	g = guest.Active()
	if g == nil || g.RootPath() == "" {
		return "", nil, false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	root, err := filepath.Abs(g.RootPath())
	if err != nil {
		root = g.RootPath()
	}
	abs = filepath.Clean(abs)
	root = filepath.Clean(root)
	if abs == root {
		return "/", g, true
	}
	prefix := root + string(filepath.Separator)
	if !strings.HasPrefix(abs, prefix) {
		return "", nil, false
	}
	rel := strings.TrimPrefix(abs, prefix)
	return guest.NormalizeGuestPath(rel), g, true
}

type guestFileInfo struct {
	name  string
	mode  os.FileMode
	isDir bool
}

func (i *guestFileInfo) Name() string       { return i.name }
func (i *guestFileInfo) Size() int64        { return 0 }
func (i *guestFileInfo) Mode() os.FileMode  { return i.mode }
func (i *guestFileInfo) ModTime() time.Time { return time.Time{} }
func (i *guestFileInfo) IsDir() bool        { return i.isDir }
func (i *guestFileInfo) Sys() any           { return nil }
