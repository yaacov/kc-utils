//go:build linux

package guest

import (
	"os"
	"path/filepath"
	"strings"
)

// FileRead reads path. Paths under the active guest RootPath go through the
// guest backend; other paths use os.ReadFile.
func FileRead(path string) ([]byte, error) {
	if gp, g, ok := guestPathFromHost(path); ok {
		return g.ReadFile(gp)
	}
	return os.ReadFile(path)
}

// FileWrite writes path via the guest backend when under RootPath.
func FileWrite(path string, data []byte, perm os.FileMode) error {
	if gp, g, ok := guestPathFromHost(path); ok {
		return g.WriteFile(gp, data, perm)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, perm)
}

// FileExists reports whether path exists (guest backend or os.Stat).
func FileExists(path string) bool {
	if gp, g, ok := guestPathFromHost(path); ok {
		return g.Exists(gp)
	}
	_, err := os.Stat(path)
	return err == nil
}

// FileIsDir reports whether path is a directory.
func FileIsDir(path string) bool {
	if gp, g, ok := guestPathFromHost(path); ok {
		return g.IsDir(gp)
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// FileStat returns file info. For guestfs paths, Mode is best-effort.
func FileStat(path string) (os.FileInfo, error) {
	if gp, g, ok := guestPathFromHost(path); ok {
		if !g.Exists(gp) {
			return nil, &os.PathError{Op: "stat", Path: path, Err: os.ErrNotExist}
		}
		isDir := g.IsDir(gp)
		mode := os.FileMode(0o644)
		if isDir {
			mode = os.ModeDir | 0o755
		}
		return &guestFileInfo{name: filepath.Base(path), mode: mode, isDir: isDir}, nil
	}
	return os.Stat(path)
}

// FileRemove removes a file via the guest backend when under RootPath.
func FileRemove(path string) error {
	if gp, g, ok := guestPathFromHost(path); ok {
		return g.Remove(gp)
	}
	return os.Remove(path)
}

// FileRemoveAll removes a path tree via the guest backend when under RootPath.
func FileRemoveAll(path string) error {
	if gp, g, ok := guestPathFromHost(path); ok {
		return g.RemoveAll(gp)
	}
	return os.RemoveAll(path)
}

// FileMkdirAll creates directories via the guest backend when under RootPath.
func FileMkdirAll(path string, perm os.FileMode) error {
	if gp, g, ok := guestPathFromHost(path); ok {
		return g.Mkdir(gp, perm)
	}
	return os.MkdirAll(path, perm)
}

// FileReadDir lists a directory.
func FileReadDir(path string) ([]DirEntry, error) {
	if gp, g, ok := guestPathFromHost(path); ok {
		return g.ReadDir(gp)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	out := make([]DirEntry, 0, len(entries))
	for _, e := range entries {
		info, _ := e.Info()
		mode := os.FileMode(0)
		if info != nil {
			mode = info.Mode()
		}
		out = append(out, DirEntry{Name: e.Name(), IsDir: e.IsDir(), Mode: mode})
	}
	return out, nil
}

// FileRename renames within the guest when both paths are under RootPath.
func FileRename(oldpath, newpath string) error {
	gpOld, g, okOld := guestPathFromHost(oldpath)
	gpNew, _, okNew := guestPathFromHost(newpath)
	if okOld && okNew {
		return g.Rename(gpOld, gpNew)
	}
	return os.Rename(oldpath, newpath)
}

// FileSymlink creates a symlink at linkpath (guest or host).
func FileSymlink(target, linkpath string) error {
	if gp, g, ok := guestPathFromHost(linkpath); ok {
		return g.Symlink(target, gp)
	}
	return os.Symlink(target, linkpath)
}

// FileChmod changes mode via the guest backend when under RootPath.
func FileChmod(path string, mode os.FileMode) error {
	if gp, g, ok := guestPathFromHost(path); ok {
		return g.Chmod(gp, mode)
	}
	return os.Chmod(path, mode)
}

// FileGlob expands a pattern. Matches under the guest root are returned as
// host-style paths under RootPath for caller compatibility.
func FileGlob(pattern string) ([]string, error) {
	if g := Active(); g != nil && g.rootPath != "" {
		root, _ := filepath.Abs(g.rootPath)
		absPat, err := filepath.Abs(pattern)
		if err == nil && (absPat == root || strings.HasPrefix(absPat, root+string(filepath.Separator))) {
			rel, _ := filepath.Rel(root, absPat)
			matches, err := g.Glob(normalizeGuestPath(rel))
			if err != nil {
				return nil, err
			}
			out := make([]string, 0, len(matches))
			for _, m := range matches {
				out = append(out, g.HostPath(m))
			}
			return out, nil
		}
	}
	return filepath.Glob(pattern)
}

// FileUpload copies a real host path into a guest path (host-style under RootPath).
func FileUpload(hostPath, guestHostStylePath string) error {
	if gp, g, ok := guestPathFromHost(guestHostStylePath); ok {
		return g.Upload(hostPath, gp)
	}
	info, err := os.Stat(hostPath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(hostPath, guestHostStylePath)
	}
	if err := os.MkdirAll(filepath.Dir(guestHostStylePath), 0o755); err != nil {
		return err
	}
	return copyFile(hostPath, guestHostStylePath, info.Mode())
}

// FileStatFS returns free bytes/inodes for the filesystem containing path.
func FileStatFS(path string) (freeBytes, freeInodes int64, err error) {
	if gp, g, ok := guestPathFromHost(path); ok {
		return g.StatFS(gp)
	}
	return hostStatFS(path)
}

// FileWalkDir walks a guest directory tree (host-style path under RootPath).
func FileWalkDir(root string, fn func(path string, isDir bool) error) error {
	info, err := FileStat(root)
	if err != nil {
		return err
	}
	if err := fn(root, info.IsDir()); err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}
	entries, err := FileReadDir(root)
	if err != nil {
		return err
	}
	for _, e := range entries {
		child := filepath.Join(root, e.Name)
		if err := FileWalkDir(child, fn); err != nil {
			return err
		}
	}
	return nil
}

// FileCopy copies between two host-style guest paths (or host->guest when src
// is outside the guest root).
func FileCopy(src, dst string) error {
	if gpDst, g, okDst := guestPathFromHost(dst); okDst {
		if gpSrc, _, okSrc := guestPathFromHost(src); okSrc {
			data, err := g.ReadFile(gpSrc)
			if err != nil {
				return err
			}
			return g.WriteFile(gpDst, data, 0o644)
		}
		return g.Upload(src, gpDst)
	}
	data, err := FileRead(src)
	if err != nil {
		return err
	}
	return FileWrite(dst, data, 0o644)
}
