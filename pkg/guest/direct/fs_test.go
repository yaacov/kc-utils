//go:build linux

package direct

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	content := []byte("test content")
	if err := os.WriteFile(src, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("got %q, want %q", got, content)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("got perm %o, want 0600", info.Mode().Perm())
	}
}

func TestCopyDir(t *testing.T) {
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(t.TempDir(), "copy")
	if err := copyDir(src, dst); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dst, "a.txt"))
	if err != nil || string(got) != "a" {
		t.Fatalf("a.txt: got %q, err=%v", got, err)
	}
	got, err = os.ReadFile(filepath.Join(dst, "sub", "b.txt"))
	if err != nil || string(got) != "b" {
		t.Fatalf("sub/b.txt: got %q, err=%v", got, err)
	}
}

func TestHostStatFS(t *testing.T) {
	dir := t.TempDir()
	free, inodes, err := hostStatFS(dir)
	if err != nil {
		t.Fatal(err)
	}
	if free <= 0 {
		t.Fatalf("expected positive free bytes, got %d", free)
	}
	if inodes < 0 {
		t.Fatalf("expected non-negative inodes, got %d", inodes)
	}
}

func TestHostPathMapping(t *testing.T) {
	dir := t.TempDir()
	b := NewMounted(nil, dir, nil)
	cases := []struct {
		guestPath string
		want      string
	}{
		{"/etc/fstab", filepath.Join(dir, "etc", "fstab")},
		{"/", dir},
		{"etc/hostname", filepath.Join(dir, "etc", "hostname")},
	}
	for _, tc := range cases {
		got := b.host(tc.guestPath)
		if got != tc.want {
			t.Errorf("host(%q) = %q, want %q", tc.guestPath, got, tc.want)
		}
	}
}

func TestBackendGlob(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "etc", "hostname"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "etc", "hosts"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	b := NewMounted(nil, dir, nil)
	matches, err := b.Glob("/etc/host*")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d: %v", len(matches), matches)
	}
	for _, m := range matches {
		if m != "/etc/hostname" && m != "/etc/hosts" {
			t.Errorf("unexpected match: %q", m)
		}
	}
}

func TestBackendUploadDownload(t *testing.T) {
	guestDir := t.TempDir()
	b := NewMounted(nil, guestDir, nil)

	hostSrc := t.TempDir()
	if err := os.WriteFile(filepath.Join(hostSrc, "file.txt"), []byte("upload"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := b.Upload(filepath.Join(hostSrc, "file.txt"), "/uploaded.txt"); err != nil {
		t.Fatal(err)
	}
	data, err := b.ReadFile("/uploaded.txt")
	if err != nil || string(data) != "upload" {
		t.Fatalf("after upload: got %q, err=%v", data, err)
	}

	hostDst := filepath.Join(t.TempDir(), "downloaded.txt")
	if err := b.Download("/uploaded.txt", hostDst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(hostDst)
	if err != nil || string(got) != "upload" {
		t.Fatalf("after download: got %q, err=%v", got, err)
	}
}
