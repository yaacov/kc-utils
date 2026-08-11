package copy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsBlockEmptyIgnoresZeroStatSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "disk.raw")

	// Populated content with st_size forced to 0 (Linux block-device Stat behavior).
	data := make([]byte, 4096)
	data[0] = 0x55 // MBR/boot signature-ish non-zero
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	empty, err := isBlockEmpty(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	if empty {
		t.Fatal("expected non-zero content to be treated as populated when Stat size is 0")
	}

	// Blank content with unknown size must still count as empty.
	if err := os.WriteFile(path, make([]byte, emptyThreshold), 0o644); err != nil {
		t.Fatal(err)
	}
	empty, err = isBlockEmpty(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !empty {
		t.Fatal("expected all-zero prefix to be empty when Stat size is 0")
	}
}

func TestIsTargetEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "disk.img")
	if err := os.WriteFile(path, make([]byte, 512), 0o644); err != nil {
		t.Fatal(err)
	}
	empty, err := isTargetEmpty(Target{Path: path, IsBlockDev: false})
	if err != nil {
		t.Fatal(err)
	}
	if !empty {
		t.Fatal("expected small file to be empty")
	}

	if err := os.WriteFile(path, append(make([]byte, emptyThreshold), 1), 0o644); err != nil {
		t.Fatal(err)
	}
	empty, err = isTargetEmpty(Target{Path: path, IsBlockDev: false})
	if err != nil {
		t.Fatal(err)
	}
	if empty {
		t.Fatal("expected large zero-padded file to not be empty (size check)")
	}

	data := make([]byte, 512)
	data[0] = 1
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	empty, err = isTargetEmpty(Target{Path: path, IsBlockDev: false})
	if err != nil {
		t.Fatal(err)
	}
	if !empty {
		t.Fatal("expected sub-threshold file to be treated as empty")
	}

	data = make([]byte, emptyThreshold)
	data[0] = 1
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	empty, err = isTargetEmpty(Target{Path: path, IsBlockDev: false})
	if err != nil {
		t.Fatal(err)
	}
	if empty {
		t.Fatal("expected populated file to not be empty")
	}
}

func TestShouldLogProgress(t *testing.T) {
	if !shouldLogProgress(0, -1) {
		t.Fatal("expected first 0%")
	}
	if shouldLogProgress(0, 0) {
		t.Fatal("expected no duplicate 0%")
	}
	if !shouldLogProgress(3, -1) {
		t.Fatal("expected first sample")
	}
	if shouldLogProgress(4, 3) {
		t.Fatal("expected throttle within 5% bucket")
	}
	if !shouldLogProgress(5, 3) {
		t.Fatal("expected new 5% bucket")
	}
	if !shouldLogProgress(100, 95) {
		t.Fatal("expected 100%")
	}
}

func TestClampConcurrency(t *testing.T) {
	if got := ClampConcurrency(0, 3); got != 3 {
		t.Fatalf("default capped to disks: got %d want 3", got)
	}
	if got := ClampConcurrency(-1, 10); got != 4 {
		t.Fatalf("default: got %d want 4", got)
	}
	if got := ClampConcurrency(1, 5); got != 1 {
		t.Fatalf("sequential: got %d want 1", got)
	}
	if got := ClampConcurrency(8, 3); got != 3 {
		t.Fatalf("cap to disks: got %d want 3", got)
	}
	if got := ClampConcurrency(2, 0); got != 1 {
		t.Fatalf("empty disks: got %d want 1", got)
	}
}
