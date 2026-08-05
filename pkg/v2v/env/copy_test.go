package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	kccopy "github.com/yaacov/kc-utils/pkg/copy"
)

func TestResolveCopySourcesFromEnv(t *testing.T) {
	cfg := &Config{
		DiskPath: "[ds] vm/a.vmdk,[ds] vm/b.vmdk",
	}
	disks, err := ResolveCopySources(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(disks) != 2 || disks[0] != "[ds] vm/a.vmdk" {
		t.Fatalf("got %v", disks)
	}
}

func TestResolveCopySourcesMissing(t *testing.T) {
	cfg := &Config{}
	if _, err := ResolveCopySources(cfg); err == nil {
		t.Fatal("expected error when no disk paths configured")
	}
}

func TestNeedsCopyFlagOnly(t *testing.T) {
	if !NeedsCopy(&Config{IsInPlace: false}) {
		t.Fatal("expected NeedsCopy true when IsInPlace=false (default copy)")
	}
	if NeedsCopy(&Config{IsInPlace: true}) {
		t.Fatal("expected NeedsCopy false when IsInPlace=true")
	}
	if !NeedsCopy(&Config{IsInPlace: false, Source: "ec2"}) {
		t.Fatal("expected NeedsCopy true for IsInPlace=false regardless of source")
	}
}

func setupCopyTestTargets(t *testing.T, empty bool) (dir string, restore func()) {
	t.Helper()
	dir = t.TempDir()
	mountDir := filepath.Join(dir, "disk0")
	if err := os.Mkdir(mountDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if !empty {
		img := filepath.Join(mountDir, "disk.img")
		data := make([]byte, copyEmptyThreshold()+1)
		data[0] = 1
		if err := os.WriteFile(img, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	restore = kccopy.SetTargetGlobs(filepath.Join(dir, "block*"), filepath.Join(dir, "disk*"))
	return dir, restore
}

// copyEmptyThreshold mirrors pkg/copy emptyThreshold (1 MiB) for test fixtures.
func copyEmptyThreshold() int { return 1 << 20 }

func TestValidateCopyModeCopyOK(t *testing.T) {
	_, restore := setupCopyTestTargets(t, true)
	defer restore()

	cfg := &Config{
		IsInPlace:  false,
		Source:     "vSphere",
		LibvirtURL: "vpx://user@vcenter/dc/host/esxi",
		VmName:     "my-vm",
	}
	if err := ValidateCopyMode(cfg); err != nil {
		t.Fatalf("expected OK: %v", err)
	}
}

func TestValidateCopyModeCopyPopulatedFails(t *testing.T) {
	_, restore := setupCopyTestTargets(t, false)
	defer restore()

	cfg := &Config{
		IsInPlace:  false,
		Source:     "vSphere",
		LibvirtURL: "vpx://user@vcenter/dc/host/esxi",
		VmName:     "my-vm",
	}
	err := ValidateCopyMode(cfg)
	if err == nil || !strings.Contains(err.Error(), "already populated") {
		t.Fatalf("expected populated mismatch error, got %v", err)
	}
}

func TestValidateCopyModeInPlaceOK(t *testing.T) {
	_, restore := setupCopyTestTargets(t, false)
	defer restore()

	cfg := &Config{IsInPlace: true}
	if err := ValidateCopyMode(cfg); err != nil {
		t.Fatalf("expected OK: %v", err)
	}
}

func TestValidateCopyModeInPlaceEmptyFails(t *testing.T) {
	_, restore := setupCopyTestTargets(t, true)
	defer restore()

	cfg := &Config{IsInPlace: true}
	err := ValidateCopyMode(cfg)
	if err == nil || !strings.Contains(err.Error(), "PVC targets are empty") {
		t.Fatalf("expected empty mismatch error, got %v", err)
	}
}

func TestValidateCopyModeCopyWrongSource(t *testing.T) {
	_, restore := setupCopyTestTargets(t, true)
	defer restore()

	cfg := &Config{
		IsInPlace:  false,
		Source:     "ec2",
		LibvirtURL: "vpx://user@vcenter/dc/host/esxi",
		VmName:     "my-vm",
	}
	err := ValidateCopyMode(cfg)
	if err == nil || !strings.Contains(err.Error(), "vSphere") {
		t.Fatalf("expected vSphere source error, got %v", err)
	}
}

func TestValidateCopyModeNoTargets(t *testing.T) {
	dir := t.TempDir()
	restore := kccopy.SetTargetGlobs(filepath.Join(dir, "block*"), filepath.Join(dir, "disk*"))
	defer restore()

	err := ValidateCopyMode(&Config{IsInPlace: true})
	if err == nil || !strings.Contains(err.Error(), "no PVC targets") {
		t.Fatalf("expected no targets error, got %v", err)
	}
}
