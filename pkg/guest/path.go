//go:build linux

package guest

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// guestPathFromHost translates a host-style path under the active guest RootPath
// into a guest-absolute path. ok is false when there is no active guest or path
// is outside the guest root (caller should use host os.*).
func guestPathFromHost(path string) (guestPath string, g *Guest, ok bool) {
	g = Active()
	if g == nil || g.rootPath == "" {
		return "", nil, false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	root, err := filepath.Abs(g.rootPath)
	if err != nil {
		root = g.rootPath
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
	return normalizeGuestPath(rel), g, true
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
