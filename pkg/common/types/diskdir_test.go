package types

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImageFileName(t *testing.T) {
	if got := ImageFileName(0); got != "disk0.img" {
		t.Fatalf("ImageFileName(0) = %q", got)
	}
	if got := ImageFileName(12); got != "disk12.img" {
		t.Fatalf("ImageFileName(12) = %q", got)
	}
}

func TestExpandDiskDir(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"disk10.img", "disk0.img", "disk2.img", "skip.img", "disk0.raw"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := ExpandDiskDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join(dir, "disk0.img"),
		filepath.Join(dir, "disk2.img"),
		filepath.Join(dir, "disk10.img"),
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%+v)", len(got), len(want), got)
	}
	for i, spec := range got {
		if spec.Path != want[i] {
			t.Errorf("path[%d] = %q, want %q", i, spec.Path, want[i])
		}
		if spec.Format != "raw" {
			t.Errorf("format[%d] = %q, want raw", i, spec.Format)
		}
	}
}

func TestExpandDiskDirEmpty(t *testing.T) {
	_, err := ExpandDiskDir("")
	if err == nil {
		t.Fatal("expected error for empty dir")
	}
}

func TestExpandDiskDirMissing(t *testing.T) {
	_, err := ExpandDiskDir(filepath.Join(t.TempDir(), "nope"))
	if err == nil {
		t.Fatal("expected error for missing dir")
	}
}

func TestExpandDiskDirNotDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ExpandDiskDir(path)
	if err == nil {
		t.Fatal("expected error for non-directory")
	}
}

func TestExpandDiskDirNoImages(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "other.img"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ExpandDiskDir(dir)
	if err == nil {
		t.Fatal("expected error when no diskN.img files exist")
	}
}

func TestExpandDiskDirIgnoresNonImages(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "dir["), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "disk0-backup.img"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "disk0.img"), 0o755); err != nil {
		t.Fatal(err)
	}
	overflow := filepath.Join(dir, "disk"+strings.Repeat("9", 40)+".img")
	if err := os.WriteFile(overflow, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "disk1.img")
	if err := os.WriteFile(want, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ExpandDiskDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Path != want || got[0].Format != "raw" {
		t.Fatalf("got %+v, want only %s", got, want)
	}
}

func TestExpandDiskDirMetacharPath(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "dir[")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "disk0.img")
	if err := os.WriteFile(want, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ExpandDiskDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Path != want {
		t.Fatalf("got %+v, want %s", got, want)
	}
}
