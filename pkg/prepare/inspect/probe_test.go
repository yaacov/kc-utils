//go:build linux

package inspect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProbeRootAmazonLinuxSymlink(t *testing.T) {
	root := t.TempDir()
	etc := filepath.Join(root, "etc")
	usrLib := filepath.Join(root, "usr", "lib")
	if err := os.MkdirAll(etc, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(usrLib, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(usrLib, "os-release"), []byte(`ID=amzn
PRETTY_NAME="Amazon Linux 2023"
VERSION_ID=2023
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../usr/lib/os-release", filepath.Join(etc, "os-release")); err != nil {
		t.Fatal(err)
	}

	data, ok := ProbeRoot(root)
	if !ok {
		t.Fatal("ProbeRoot returned false, want true for Amazon Linux symlink layout")
	}
	if data.Distro != "amzn" {
		t.Errorf("distro = %q, want amzn", data.Distro)
	}
	if data.ProductName != "Amazon Linux 2023" {
		t.Errorf("product = %q, want Amazon Linux 2023", data.ProductName)
	}
	if data.MajorVersion != 2023 {
		t.Errorf("major version = %d, want 2023", data.MajorVersion)
	}
}

func TestProbeRootBrokenSymlinkOnly(t *testing.T) {
	root := t.TempDir()
	etc := filepath.Join(root, "etc")
	if err := os.MkdirAll(etc, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../usr/lib/os-release", filepath.Join(etc, "os-release")); err != nil {
		t.Fatal(err)
	}

	_, ok := ProbeRoot(root)
	if ok {
		t.Fatal("ProbeRoot returned true, want false for broken etc/os-release symlink only")
	}
}

func TestProbeRootUsrLibOSReleaseOnly(t *testing.T) {
	root := t.TempDir()
	usrLib := filepath.Join(root, "usr", "lib")
	if err := os.MkdirAll(usrLib, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(usrLib, "os-release"), []byte("ID=fedora\nVERSION_ID=39\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	data, ok := ProbeRoot(root)
	if !ok {
		t.Fatal("ProbeRoot returned false, want true for usr/lib/os-release only")
	}
	if data.Distro != "fedora" {
		t.Errorf("distro = %q, want fedora", data.Distro)
	}
}
