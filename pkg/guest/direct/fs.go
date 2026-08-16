//go:build linux

package direct

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
	guestcommon "github.com/yaacov/kc-utils/pkg/guest/common"
)

func (b *Backend) host(guestPath string) string {
	return filepath.Join(b.mountRoot, strings.TrimPrefix(filepath.Clean("/"+filepath.ToSlash(guestPath)), "/"))
}

func (b *Backend) ReadFile(guestPath string) ([]byte, error) {
	return os.ReadFile(b.host(guestPath))
}

func (b *Backend) WriteFile(guestPath string, data []byte, perm os.FileMode) error {
	p := b.host(guestPath)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, perm)
}

func (b *Backend) Exists(guestPath string) bool {
	_, err := os.Stat(b.host(guestPath))
	return err == nil
}

func (b *Backend) IsDir(guestPath string) bool {
	info, err := os.Stat(b.host(guestPath))
	return err == nil && info.IsDir()
}

func (b *Backend) Glob(pattern string) ([]string, error) {
	matches, err := filepath.Glob(b.host(pattern))
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		gp, err := filepath.Rel(b.mountRoot, m)
		if err != nil {
			continue
		}
		out = append(out, "/"+filepath.ToSlash(gp))
	}
	return out, nil
}

func (b *Backend) Remove(guestPath string) error {
	return os.Remove(b.host(guestPath))
}

func (b *Backend) RemoveAll(guestPath string) error {
	return os.RemoveAll(b.host(guestPath))
}

func (b *Backend) Rename(oldPath, newPath string) error {
	return os.Rename(b.host(oldPath), b.host(newPath))
}

func (b *Backend) Symlink(target, link string) error {
	return os.Symlink(target, b.host(link))
}

func (b *Backend) Readlink(guestPath string) (string, error) {
	return os.Readlink(b.host(guestPath))
}

func (b *Backend) Chmod(guestPath string, mode os.FileMode) error {
	return os.Chmod(b.host(guestPath), mode)
}

func (b *Backend) MkdirAll(guestPath string, perm os.FileMode) error {
	return os.MkdirAll(b.host(guestPath), perm)
}

func (b *Backend) ReadDir(guestPath string) ([]types.GuestDirEntry, error) {
	entries, err := os.ReadDir(b.host(guestPath))
	if err != nil {
		return nil, err
	}
	out := make([]types.GuestDirEntry, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		mode := os.FileMode(0)
		if err == nil {
			mode = info.Mode()
		}
		out = append(out, types.GuestDirEntry{Name: e.Name(), IsDir: e.IsDir(), Mode: mode})
	}
	return out, nil
}

func (b *Backend) Upload(hostPath, guestPath string) error {
	info, err := os.Stat(hostPath)
	if err != nil {
		return err
	}
	dst := b.host(guestPath)
	if info.IsDir() {
		return guestcommon.CopyDir(hostPath, dst)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return guestcommon.CopyFile(hostPath, dst, info.Mode())
}

func (b *Backend) Download(guestPath, hostPath string) error {
	src := b.host(guestPath)
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return guestcommon.CopyDir(src, hostPath)
	}
	if err := os.MkdirAll(filepath.Dir(hostPath), 0o755); err != nil {
		return err
	}
	return guestcommon.CopyFile(src, hostPath, info.Mode())
}

func (b *Backend) StatFS(guestPath string) (freeBytes, freeInodes int64, err error) {
	return guestcommon.HostStatFS(b.host(guestPath))
}
