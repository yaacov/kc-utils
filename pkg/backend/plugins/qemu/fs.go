//go:build unix

package qemu

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func (b *Backend) ReadFile(guestPath string) ([]byte, error) {
	data, err := b.session.client.readFile(guestToAppliance(guestPath))
	if err != nil {
		return nil, pathError("read", guestPath, err)
	}
	return data, nil
}

func (b *Backend) WriteFile(guestPath string, data []byte, perm os.FileMode) error {
	ap := guestToAppliance(guestPath)
	parent := path.Dir(ap)
	if parent != "/" && parent != "." {
		if err := b.session.client.mkdirAll(parent, 0o755); err != nil {
			return pathError("write", guestPath, err)
		}
	}
	if err := b.session.client.writeFile(ap, data, perm); err != nil {
		return pathError("write", guestPath, err)
	}
	return nil
}

func (b *Backend) Exists(guestPath string) bool {
	st, err := b.session.client.stat(guestToAppliance(guestPath))
	return err == nil && st.Exists
}

func (b *Backend) IsDir(guestPath string) bool {
	st, err := b.session.client.stat(guestToAppliance(guestPath))
	return err == nil && st.Exists && st.IsDir
}

// Glob matches guest-absolute paths. It walks the deepest literal prefix of the
// pattern via ReadDir and matches the wildcard segments host-side.
func (b *Backend) Glob(pattern string) ([]string, error) {
	searchRoot, patternSuffix := globSplit(pattern)
	var matches []string
	err := b.walk(searchRoot, func(guestPath string, _ bool) {
		rel := strings.TrimPrefix(strings.TrimPrefix(guestPath, searchRoot), "/")
		if rel == "" {
			return
		}
		if globMatchSegments(rel, patternSuffix) {
			matches = append(matches, guestPath)
		}
	})
	if err != nil {
		return nil, pathError("glob", pattern, err)
	}
	return matches, nil
}

// walk visits every path under root (guest-absolute), calling fn with each
// path and whether it is a directory. It descends into subdirectories.
func (b *Backend) walk(root string, fn func(guestPath string, isDir bool)) error {
	entries, err := b.session.client.readDir(guestToAppliance(root))
	if err != nil {
		return nil // missing directory yields no matches, not an error
	}
	for _, e := range entries {
		child := path.Join(root, e.Name)
		fn(child, e.IsDir)
		if e.IsDir {
			if err := b.walk(child, fn); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *Backend) Remove(guestPath string) error {
	if err := b.session.client.remove(guestToAppliance(guestPath), false); err != nil {
		return pathError("remove", guestPath, err)
	}
	return nil
}

func (b *Backend) RemoveAll(guestPath string) error {
	if err := b.session.client.remove(guestToAppliance(guestPath), true); err != nil {
		return pathError("removeall", guestPath, err)
	}
	return nil
}

func (b *Backend) Rename(oldPath, newPath string) error {
	if err := b.session.client.rename(guestToAppliance(oldPath), guestToAppliance(newPath)); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", oldPath, newPath, err)
	}
	return nil
}

// Symlink creates link pointing at target. target is stored verbatim (guest
// paths inside a symlink stay guest-absolute); only link is rebased.
func (b *Backend) Symlink(target, link string) error {
	if err := b.session.client.symlink(target, guestToAppliance(link)); err != nil {
		return pathError("symlink", link, err)
	}
	return nil
}

func (b *Backend) Readlink(guestPath string) (string, error) {
	target, err := b.session.client.readlink(guestToAppliance(guestPath))
	if err != nil {
		return "", pathError("readlink", guestPath, err)
	}
	return target, nil
}

func (b *Backend) Chmod(guestPath string, mode os.FileMode) error {
	if err := b.session.client.chmod(guestToAppliance(guestPath), mode); err != nil {
		return pathError("chmod", guestPath, err)
	}
	return nil
}

func (b *Backend) MkdirAll(guestPath string, perm os.FileMode) error {
	if guestPath == "/" {
		return nil
	}
	if err := b.session.client.mkdirAll(guestToAppliance(guestPath), perm); err != nil {
		return pathError("mkdir", guestPath, err)
	}
	return nil
}

func (b *Backend) ReadDir(guestPath string) ([]types.GuestDirEntry, error) {
	entries, err := b.session.client.readDir(guestToAppliance(guestPath))
	if err != nil {
		return nil, pathError("readdir", guestPath, err)
	}
	out := make([]types.GuestDirEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, types.GuestDirEntry{
			Name:  e.Name,
			IsDir: e.IsDir,
			Mode:  os.FileMode(e.Mode),
		})
	}
	return out, nil
}

// Upload copies a host file or directory into the guest at guestPath.
func (b *Backend) Upload(hostPath, guestPath string) error {
	info, err := os.Stat(hostPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		data, err := os.ReadFile(hostPath)
		if err != nil {
			return err
		}
		return b.WriteFile(guestPath, data, info.Mode().Perm())
	}
	// Directory: replace destination and copy entries recursively.
	if err := b.RemoveAll(guestPath); err != nil {
		return pathError("upload", guestPath, err)
	}
	return filepath.Walk(hostPath, func(p string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(hostPath, p)
		if err != nil {
			return err
		}
		dest := path.Join(guestPath, filepath.ToSlash(rel))
		if fi.IsDir() {
			return b.MkdirAll(dest, fi.Mode().Perm())
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return b.WriteFile(dest, data, fi.Mode().Perm())
	})
}

// Download copies a guest file or directory to a host path.
func (b *Backend) Download(guestPath, hostPath string) error {
	st, err := b.session.client.stat(guestToAppliance(guestPath))
	if err != nil {
		return pathError("download", guestPath, err)
	}
	if !st.Exists {
		return pathError("download", guestPath, os.ErrNotExist)
	}
	if !st.IsDir {
		if err := os.MkdirAll(filepath.Dir(hostPath), 0o755); err != nil {
			return err
		}
		data, err := b.session.client.readFile(guestToAppliance(guestPath))
		if err != nil {
			return pathError("download", guestPath, err)
		}
		return os.WriteFile(hostPath, data, st.Mode.Perm())
	}
	if err := os.MkdirAll(hostPath, 0o755); err != nil {
		return err
	}
	entries, err := b.session.client.readDir(guestToAppliance(guestPath))
	if err != nil {
		return pathError("download", guestPath, err)
	}
	for _, e := range entries {
		if err := b.Download(path.Join(guestPath, e.Name), filepath.Join(hostPath, e.Name)); err != nil {
			return err
		}
	}
	return nil
}

func (b *Backend) StatFS(guestPath string) (freeBytes, freeInodes int64, err error) {
	fb, fi, err := b.session.client.statFS(guestToAppliance(guestPath))
	if err != nil {
		return 0, 0, pathError("statfs", guestPath, err)
	}
	return fb, fi, nil
}

// pathError wraps err with the operation and guest path for context.
func pathError(op, guestPath string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s %s: %w", op, guestPath, err)
}

// globSplit splits a glob pattern into the deepest literal directory prefix and
// the remaining wildcard segments.
func globSplit(pattern string) (root, suffix string) {
	parts := strings.Split(pattern, "/")
	i := 0
	for i < len(parts) {
		if strings.ContainsAny(parts[i], "*?[") {
			break
		}
		i++
	}
	if i == 0 {
		return "/", pattern
	}
	return strings.Join(parts[:i], "/"), strings.Join(parts[i:], "/")
}

// globMatchSegments matches a relative path against a multi-segment glob.
func globMatchSegments(relPath, pattern string) bool {
	relParts := strings.Split(relPath, "/")
	patParts := strings.Split(pattern, "/")
	if len(relParts) != len(patParts) {
		return false
	}
	for i, pat := range patParts {
		matched, err := path.Match(pat, relParts[i])
		if err != nil || !matched {
			return false
		}
	}
	return true
}
