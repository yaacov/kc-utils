//go:build linux

package guestfs

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

func (b *Backend) ensureMounted() error {
	if err := b.ensureSessionWithRecovery(); err != nil {
		return err
	}
	if b.mountsActive {
		return nil
	}
	specs, err := b.effectiveMountSpecs()
	if err != nil {
		return err
	}
	script := mountScriptPrefix(specs)
	if _, err := b.session.remoteScript(script); err != nil {
		return fmt.Errorf("mounting guest filesystems: %w", err)
	}
	b.mountsActive = true
	return nil
}

func (b *Backend) withMounted(scriptBody string) (string, error) {
	if err := b.ensureMounted(); err != nil {
		return "", err
	}
	out, err := b.withRecovery(func() ([]byte, error) {
		return b.session.remoteScript(scriptBody)
	})
	return string(out), err
}

func (b *Backend) ReadFile(guestPath string) ([]byte, error) {
	tmp, err := os.CreateTemp("", "kc-gfs-read-*")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	var body strings.Builder
	body.WriteString("download ")
	body.WriteString(quoteGuestfish(guestPath))
	body.WriteByte(' ')
	body.WriteString(quoteGuestfish(tmpPath))
	body.WriteByte('\n')
	if _, err := b.withMounted(body.String()); err != nil {
		return nil, pathError("read", guestPath, err)
	}
	return os.ReadFile(tmpPath)
}

func (b *Backend) WriteFile(guestPath string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp("", "kc-gfs-write-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	tmp.Close()
	defer os.Remove(tmpPath)

	parent := path.Dir(guestPath)
	var body strings.Builder
	if parent != "/" && parent != "." {
		body.WriteString("mkdir-p ")
		body.WriteString(quoteGuestfish(parent))
		body.WriteByte('\n')
	}
	body.WriteString("upload ")
	body.WriteString(quoteGuestfish(tmpPath))
	body.WriteByte(' ')
	body.WriteString(quoteGuestfish(guestPath))
	body.WriteByte('\n')
	body.WriteString("chmod 0")
	body.WriteString(strconv.FormatUint(uint64(perm.Perm()), 8))
	body.WriteByte(' ')
	body.WriteString(quoteGuestfish(guestPath))
	body.WriteByte('\n')
	_, err = b.withMounted(body.String())
	return pathError("write", guestPath, err)
}

func (b *Backend) Exists(guestPath string) bool {
	var body strings.Builder
	body.WriteString("-exists ")
	body.WriteString(quoteGuestfish(guestPath))
	body.WriteByte('\n')
	out, err := b.withMounted(body.String())
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(out), "true")
}

func (b *Backend) IsDir(guestPath string) bool {
	var body strings.Builder
	body.WriteString("-is-dir ")
	body.WriteString(quoteGuestfish(guestPath))
	body.WriteByte('\n')
	out, err := b.withMounted(body.String())
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(out), "true")
}

func (b *Backend) Glob(pattern string) ([]string, error) {
	searchRoot, patternSuffix := globSplit(pattern)

	var body strings.Builder
	body.WriteString("find ")
	body.WriteString(quoteGuestfish(searchRoot))
	body.WriteByte('\n')
	out, err := b.withMounted(body.String())
	if err != nil {
		return nil, pathError("glob", pattern, err)
	}
	var matches []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		rel := strings.TrimPrefix(line, "/")
		if globMatchSegments(rel, patternSuffix) {
			matches = append(matches, path.Join(searchRoot, line))
		}
	}
	return matches, nil
}

// globSplit splits a glob pattern into the deepest literal directory prefix
// and the remaining pattern segments containing wildcards.
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

// globMatchSegments matches a relative path against a multi-segment glob
// pattern (e.g. "virtio/virtio_blk.ko.xz" against "*/*.ko*").
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

func (b *Backend) Remove(guestPath string) error {
	var body strings.Builder
	body.WriteString("-rm ")
	body.WriteString(quoteGuestfish(guestPath))
	body.WriteByte('\n')
	_, err := b.withMounted(body.String())
	return pathError("remove", guestPath, err)
}

func (b *Backend) RemoveAll(guestPath string) error {
	var body strings.Builder
	body.WriteString("-rm-rf ")
	body.WriteString(quoteGuestfish(guestPath))
	body.WriteByte('\n')
	_, err := b.withMounted(body.String())
	return pathError("removeall", guestPath, err)
}

func (b *Backend) Rename(oldPath, newPath string) error {
	var body strings.Builder
	body.WriteString("mv ")
	body.WriteString(quoteGuestfish(oldPath))
	body.WriteByte(' ')
	body.WriteString(quoteGuestfish(newPath))
	body.WriteByte('\n')
	_, err := b.withMounted(body.String())
	if err != nil {
		return fmt.Errorf("rename %s -> %s: %w", oldPath, newPath, err)
	}
	return nil
}

func (b *Backend) Symlink(target, link string) error {
	var body strings.Builder
	body.WriteString("ln-s ")
	body.WriteString(quoteGuestfish(target))
	body.WriteByte(' ')
	body.WriteString(quoteGuestfish(link))
	body.WriteByte('\n')
	_, err := b.withMounted(body.String())
	return pathError("symlink", link, err)
}

func (b *Backend) Readlink(guestPath string) (string, error) {
	var body strings.Builder
	body.WriteString("readlink ")
	body.WriteString(quoteGuestfish(guestPath))
	body.WriteByte('\n')
	out, err := b.withMounted(body.String())
	if err != nil {
		return "", pathError("readlink", guestPath, err)
	}
	target := strings.TrimSuffix(out, "\n")
	if target == "" {
		return "", fmt.Errorf("readlink %s: empty target", guestPath)
	}
	return target, nil
}

func (b *Backend) Chmod(guestPath string, mode os.FileMode) error {
	var body strings.Builder
	body.WriteString("chmod 0")
	body.WriteString(strconv.FormatUint(uint64(mode.Perm()), 8))
	body.WriteByte(' ')
	body.WriteString(quoteGuestfish(guestPath))
	body.WriteByte('\n')
	_, err := b.withMounted(body.String())
	return pathError("chmod", guestPath, err)
}

func (b *Backend) MkdirAll(guestPath string, _ os.FileMode) error {
	if guestPath == "/" {
		return nil
	}
	var body strings.Builder
	body.WriteString("mkdir-p ")
	body.WriteString(quoteGuestfish(guestPath))
	body.WriteByte('\n')
	_, err := b.withMounted(body.String())
	return pathError("mkdir", guestPath, err)
}

func (b *Backend) ReadDir(guestPath string) ([]types.GuestDirEntry, error) {
	// Guard: -is-dir returns cleanly for non-existent paths, while ll can
	// crash the guestfish daemon.
	if !b.IsDir(guestPath) {
		return nil, pathError("readdir", guestPath, os.ErrNotExist)
	}
	var body strings.Builder
	body.WriteString("ll ")
	body.WriteString(quoteGuestfish(guestPath))
	body.WriteByte('\n')
	out, err := b.withMounted(body.String())
	if err != nil {
		return nil, pathError("readdir", guestPath, err)
	}
	var entries []types.GuestDirEntry
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0] == 't' {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}
		name := strings.Join(fields[8:], " ")
		if name == "." || name == ".." {
			continue
		}
		entries = append(entries, types.GuestDirEntry{
			Name:  name,
			IsDir: line[0] == 'd',
		})
	}
	return entries, nil
}

func (b *Backend) Upload(hostPath, guestPath string) error {
	info, err := os.Stat(hostPath)
	if err != nil {
		return err
	}
	var body strings.Builder
	parent := path.Dir(guestPath)
	if parent != "/" && parent != "." {
		body.WriteString("mkdir-p ")
		body.WriteString(quoteGuestfish(parent))
		body.WriteByte('\n')
	}
	if info.IsDir() {
		body.WriteString("copy-in ")
		body.WriteString(quoteGuestfish(hostPath))
		body.WriteByte(' ')
		body.WriteString(quoteGuestfish(parent))
		body.WriteByte('\n')
		base := filepath.Base(hostPath)
		placed := path.Join(parent, base)
		if placed != guestPath {
			body.WriteString("mv ")
			body.WriteString(quoteGuestfish(placed))
			body.WriteByte(' ')
			body.WriteString(quoteGuestfish(guestPath))
			body.WriteByte('\n')
		}
	} else {
		body.WriteString("upload ")
		body.WriteString(quoteGuestfish(hostPath))
		body.WriteByte(' ')
		body.WriteString(quoteGuestfish(guestPath))
		body.WriteByte('\n')
	}
	_, err = b.withMounted(body.String())
	return pathError("upload", guestPath, err)
}

func (b *Backend) StatFS(guestPath string) (freeBytes, freeInodes int64, err error) {
	var body strings.Builder
	body.WriteString("statvfs ")
	body.WriteString(quoteGuestfish(guestPath))
	body.WriteByte('\n')
	out, err := b.withMounted(body.String())
	if err != nil {
		return 0, 0, pathError("statfs", guestPath, err)
	}
	var bsize, bavail, files, ffree int64
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		n, _ := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
		switch strings.TrimSpace(key) {
		case "bsize", "frsize":
			if bsize == 0 {
				bsize = n
			}
		case "bavail":
			bavail = n
		case "files":
			files = n
		case "ffree":
			ffree = n
		}
	}
	if bsize == 0 {
		bsize = 4096
	}
	// FAT/exFAT filesystems report files=0 and ffree=0 because they have no
	// inode concept. Signal "not applicable" with -1 so callers don't treat
	// it as "exhausted."
	if files == 0 && ffree == 0 {
		ffree = -1
	}
	return bavail * bsize, ffree, nil
}

func (b *Backend) Download(guestPath, hostPath string) error {
	if err := os.MkdirAll(filepath.Dir(hostPath), 0o755); err != nil {
		return err
	}
	var body strings.Builder
	if b.IsDir(guestPath) {
		if err := os.MkdirAll(hostPath, 0o755); err != nil {
			return err
		}
		body.WriteString("copy-out ")
		body.WriteString(quoteGuestfish(guestPath))
		body.WriteByte(' ')
		body.WriteString(quoteGuestfish(filepath.Dir(hostPath)))
		body.WriteByte('\n')
		_, err := b.withMounted(body.String())
		if err != nil {
			return pathError("download", guestPath, err)
		}
		placed := filepath.Join(filepath.Dir(hostPath), path.Base(guestPath))
		if placed != hostPath {
			_ = os.RemoveAll(hostPath)
			return os.Rename(placed, hostPath)
		}
		return nil
	}
	body.WriteString("download ")
	body.WriteString(quoteGuestfish(guestPath))
	body.WriteByte(' ')
	body.WriteString(quoteGuestfish(hostPath))
	body.WriteByte('\n')
	_, err := b.withMounted(body.String())
	return pathError("download", guestPath, err)
}
