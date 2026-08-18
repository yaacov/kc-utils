//go:build unix

package native

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyHostname(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}

	c := &NativeCustomizer{}
	opts := map[string]string{"hostname": "myhost.example.com"}
	if err := c.Apply(root, opts); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(root, "etc", "hostname"))
	if err != nil {
		t.Fatalf("reading hostname file: %v", err)
	}
	if string(got) != "myhost.example.com\n" {
		t.Errorf("hostname file content = %q, want %q", string(got), "myhost.example.com\n")
	}
}

func TestApplyTimezone(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}

	c := &NativeCustomizer{}
	opts := map[string]string{"timezone": "Europe/Prague"}
	if err := c.Apply(root, opts); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	link := filepath.Join(root, "etc", "localtime")
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("reading symlink: %v", err)
	}
	if target != "/usr/share/zoneinfo/Europe/Prague" {
		t.Errorf("localtime symlink target = %q, want %q", target, "/usr/share/zoneinfo/Europe/Prague")
	}
}

func TestApplyEmptyOptions(t *testing.T) {
	root := t.TempDir()

	c := &NativeCustomizer{}
	if err := c.Apply(root, map[string]string{}); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no files created, found %d entries", len(entries))
	}
}

func TestApplyAutorelabelSELinux(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "selinux"), 0o755); err != nil {
		t.Fatal(err)
	}

	c := &NativeCustomizer{}
	if err := c.Apply(root, map[string]string{}); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	autorelabel := filepath.Join(root, ".autorelabel")
	if _, err := os.Stat(autorelabel); err != nil {
		t.Errorf("expected .autorelabel to be created for SELinux guest, got error: %v", err)
	}
}

func TestApplyNoAutorelabelWithoutSELinux(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}

	c := &NativeCustomizer{}
	if err := c.Apply(root, map[string]string{}); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	autorelabel := filepath.Join(root, ".autorelabel")
	if _, err := os.Stat(autorelabel); err == nil {
		t.Error("expected .autorelabel NOT to be created when etc/selinux is absent")
	}
}

func TestApplySkipAutorelabelWhenOfflineRelabeled(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "selinux"), 0o755); err != nil {
		t.Fatal(err)
	}

	c := &NativeCustomizer{}
	opts := map[string]string{"selinux_relabeled": "true"}
	if err := c.Apply(root, opts); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	autorelabel := filepath.Join(root, ".autorelabel")
	if _, err := os.Stat(autorelabel); err == nil {
		t.Error("expected .autorelabel NOT to be created when offline relabel was performed")
	}
}

func TestApplyBothOptions(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}

	c := &NativeCustomizer{}
	opts := map[string]string{
		"hostname": "dual.test",
		"timezone": "US/Eastern",
	}
	if err := c.Apply(root, opts); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(root, "etc", "hostname"))
	if err != nil {
		t.Fatalf("reading hostname file: %v", err)
	}
	if string(got) != "dual.test\n" {
		t.Errorf("hostname = %q, want %q", string(got), "dual.test\n")
	}

	target, err := os.Readlink(filepath.Join(root, "etc", "localtime"))
	if err != nil {
		t.Fatalf("reading localtime symlink: %v", err)
	}
	if target != "/usr/share/zoneinfo/US/Eastern" {
		t.Errorf("localtime target = %q, want %q", target, "/usr/share/zoneinfo/US/Eastern")
	}
}
