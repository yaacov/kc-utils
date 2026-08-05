//go:build linux

package direct

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/yaacov/kc-utils/pkg/common/types"
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
		return copyDir(hostPath, dst)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return copyFile(hostPath, dst, info.Mode())
}

func (b *Backend) Download(guestPath, hostPath string) error {
	src := b.host(guestPath)
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, hostPath)
	}
	if err := os.MkdirAll(filepath.Dir(hostPath), 0o755); err != nil {
		return err
	}
	return copyFile(src, hostPath, info.Mode())
}

func (b *Backend) StatFS(guestPath string) (freeBytes, freeInodes int64, err error) {
	return hostStatFS(b.host(guestPath))
}

// hostStatFS, copyFile, copyDir are duplicated in the parent guest_util.go to avoid an import cycle.
func hostStatFS(path string) (freeBytes, freeInodes int64, err error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0, err
	}
	ffree := int64(st.Ffree)
	// FAT/exFAT report Files=0 and Ffree=0 (no inode concept). Signal
	// "not applicable" with -1 so callers don't treat it as "exhausted."
	if st.Files == 0 && st.Ffree == 0 {
		ffree = -1
	}
	return int64(st.Bavail) * int64(st.Bsize), ffree, nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		return copyFile(path, target, info.Mode())
	})
}
