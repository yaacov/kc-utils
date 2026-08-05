//go:build linux

package standard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupFile(t *testing.T, root, relPath, content string) {
	t.Helper()
	p := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRemapFstabSDToVD(t *testing.T) {
	root := t.TempDir()
	setupFile(t, root, "etc/fstab", "/dev/sda1 / ext4 defaults 0 1\n")

	r := &Remapper{}
	if err := r.Remap(root); err != nil {
		t.Fatalf("Remap error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "etc", "fstab"))
	if !strings.Contains(string(data), "/dev/vda1") {
		t.Errorf("expected /dev/vda1 in fstab, got:\n%s", data)
	}
}

func TestRemapFstabXVDToVD(t *testing.T) {
	root := t.TempDir()
	setupFile(t, root, "etc/fstab", "/dev/xvda1 / ext4 defaults 0 1\n")

	r := &Remapper{}
	if err := r.Remap(root); err != nil {
		t.Fatalf("Remap error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "etc", "fstab"))
	if !strings.Contains(string(data), "/dev/vda1") {
		t.Errorf("expected /dev/vda1 in fstab, got:\n%s", data)
	}
}

func TestRemapCrypttab(t *testing.T) {
	root := t.TempDir()
	setupFile(t, root, "etc/fstab", "# empty\n")
	setupFile(t, root, "etc/crypttab", "luks-root /dev/sda2 none\n")

	r := &Remapper{}
	if err := r.Remap(root); err != nil {
		t.Fatalf("Remap error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "etc", "crypttab"))
	if !strings.Contains(string(data), "/dev/vda2") {
		t.Errorf("expected /dev/vda2 in crypttab, got:\n%s", data)
	}
}

func TestRemapMissingFiles(t *testing.T) {
	root := t.TempDir()
	r := &Remapper{}
	if err := r.Remap(root); err != nil {
		t.Fatalf("Remap should not error on missing files: %v", err)
	}
}

func TestName(t *testing.T) {
	r := &Remapper{}
	if r.Name() != "standard" {
		t.Errorf("Name() = %q, want standard", r.Name())
	}
}

func TestDetectPresent(t *testing.T) {
	root := t.TempDir()
	setupFile(t, root, "etc/fstab", "/dev/sda1 / ext4 defaults 0 1\n")

	r := &Remapper{}
	if !r.Detect(root) {
		t.Error("Detect = false, want true when fstab exists")
	}
}

func TestDetectAbsent(t *testing.T) {
	root := t.TempDir()
	r := &Remapper{}
	if r.Detect(root) {
		t.Error("Detect = true, want false when fstab missing")
	}
}

func TestRemapHDToVD(t *testing.T) {
	root := t.TempDir()
	setupFile(t, root, "etc/fstab", "/dev/hda1 / ext4 defaults 0 1\n")

	r := &Remapper{}
	if err := r.Remap(root); err != nil {
		t.Fatalf("Remap error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "etc", "fstab"))
	if !strings.Contains(string(data), "/dev/vda1") {
		t.Errorf("expected /dev/vda1 in fstab, got:\n%s", data)
	}
}

func TestRemapCCISSPrefix(t *testing.T) {
	root := t.TempDir()
	setupFile(t, root, "etc/fstab", "/dev/cciss/c0d0p1 / ext4 defaults 0 1\n")

	r := &Remapper{}
	if err := r.Remap(root); err != nil {
		t.Fatalf("Remap error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "etc", "fstab"))
	if !strings.Contains(string(data), "/dev/vda1") {
		t.Errorf("expected /dev/vda1 in fstab, got:\n%s", data)
	}
}

func TestRemapNVMePrefix(t *testing.T) {
	root := t.TempDir()
	setupFile(t, root, "etc/fstab", "/dev/nvme0n1p1 /boot ext4 defaults 0 1\n")

	r := &Remapper{}
	if err := r.Remap(root); err != nil {
		t.Fatalf("Remap error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "etc", "fstab"))
	if !strings.Contains(string(data), "/dev/vda1") {
		t.Errorf("expected /dev/vda1 in fstab, got:\n%s", data)
	}
}

func TestRemapGrubDefaultsKernelArgs(t *testing.T) {
	root := t.TempDir()
	setupFile(t, root, "etc/fstab", "/dev/sda1 / ext4 defaults 0 1\n")
	setupFile(t, root, "etc/default/grub", "GRUB_CMDLINE_LINUX=\"root=/dev/sda1 resume=/dev/sda2 quiet\"\nGRUB_CMDLINE_LINUX_DEFAULT=\"resume=/dev/xvda3 splash\"\n")

	r := &Remapper{}
	if err := r.Remap(root); err != nil {
		t.Fatalf("Remap error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(root, "etc", "default", "grub"))
	content := string(data)
	if !strings.Contains(content, "root=/dev/vda1") {
		t.Errorf("expected remapped root in grub config, got:\n%s", content)
	}
	if !strings.Contains(content, "resume=/dev/vda2") {
		t.Errorf("expected remapped resume in grub config, got:\n%s", content)
	}
	if !strings.Contains(content, "resume=/dev/vda3") {
		t.Errorf("expected remapped default resume in grub config, got:\n%s", content)
	}
}

func TestRemapBLSEntryOptions(t *testing.T) {
	root := t.TempDir()
	setupFile(t, root, "etc/fstab", "/dev/sda1 / ext4 defaults 0 1\n")
	setupFile(t, root, "boot/loader/entries/test.conf", "title Test\nlinux /vmlinuz\noptions root=/dev/sda1 resume=/dev/xvda2 quiet\n")

	r := &Remapper{}
	if err := r.Remap(root); err != nil {
		t.Fatalf("Remap error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(root, "boot", "loader", "entries", "test.conf"))
	content := string(data)
	if !strings.Contains(content, "root=/dev/vda1") {
		t.Errorf("expected remapped root in BLS entry, got:\n%s", content)
	}
	if !strings.Contains(content, "resume=/dev/vda2") {
		t.Errorf("expected remapped resume in BLS entry, got:\n%s", content)
	}
}
