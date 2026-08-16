//go:build linux

package common

import (
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// HostStatFS returns free bytes and free inodes for a host path.
// FAT/exFAT report Files=0 and Ffree=0 (no inode concept); freeInodes is -1
// so callers don't treat that as "exhausted."
func HostStatFS(path string) (freeBytes, freeInodes int64, err error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0, err
	}
	ffree := int64(st.Ffree)
	if st.Files == 0 && st.Ffree == 0 {
		ffree = -1
	}
	return int64(st.Bavail) * int64(st.Bsize), ffree, nil
}

// CopyFile copies a single file from src to dst with the given mode.
func CopyFile(src, dst string, mode os.FileMode) error {
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

// CopyDir recursively copies src into dst.
func CopyDir(src, dst string) error {
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
		return CopyFile(path, target, info.Mode())
	})
}
