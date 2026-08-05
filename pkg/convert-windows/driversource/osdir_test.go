package driversource

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindBestOSDirExactMatch(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "2k22"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := FindBestOSDir(base, "Windows Server 2022")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "2k22" {
		t.Fatalf("got %q, want 2k22", got)
	}
}

func TestFindBestOSDirWithPrefsPrefers2k8(t *testing.T) {
	base := t.TempDir()
	for _, dir := range []string{"2k8", "2k8R2"} {
		if err := os.MkdirAll(filepath.Join(base, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	got, err := FindBestOSDirWithPrefs(base, "Windows Server 2008 Enterprise", []string{"2k8"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "2k8" {
		t.Fatalf("got %q, want 2k8", got)
	}
}

func TestFindBestOSDirWithPrefsFallback2k8To2k8R2(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "2k8R2"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := FindBestOSDirWithPrefs(base, "Windows Server 2008 Enterprise", []string{"2k8"}, []string{"2k8R2"})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "2k8R2" {
		t.Fatalf("got %q, want 2k8R2", got)
	}
}

func TestFindBestOSDirNoCrossVersionFallback(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "2k8R2"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := FindBestOSDir(base, "Windows Server 2008 Enterprise"); err == nil {
		t.Fatal("expected error when only 2k8R2 is present for Server 2008")
	}
}

func TestFindBestOSDirMissing(t *testing.T) {
	base := t.TempDir()
	if _, err := FindBestOSDir(base, "Windows XP"); err == nil {
		t.Fatal("expected error for missing OS dir")
	}
}
