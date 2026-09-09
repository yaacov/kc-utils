package copy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverTargetsAllMounts(t *testing.T) {
	dir := t.TempDir()
	blockPath := filepath.Join(dir, "block0")
	if err := os.WriteFile(blockPath, []byte{0}, 0o644); err != nil {
		t.Fatal(err)
	}
	mountDir := filepath.Join(dir, "disk1")
	if err := os.Mkdir(mountDir, 0o755); err != nil {
		t.Fatal(err)
	}
	img := filepath.Join(mountDir, "disk.img")
	if err := os.WriteFile(img, make([]byte, 1<<20+1), 0o644); err != nil {
		t.Fatal(err)
	}

	restore := SetTargetGlobs(filepath.Join(dir, "block*"), filepath.Join(dir, "disk*"))
	defer restore()

	targets, err := DiscoverTargets()
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 2 {
		t.Fatalf("len = %d, want 2 (all discovered mounts)", len(targets))
	}
	if targets[0].Path != blockPath || !targets[0].IsBlockDev {
		t.Fatalf("block target: %+v", targets[0])
	}
	if targets[1].Path != img || targets[1].IsBlockDev {
		t.Fatalf("filesystem target: %+v", targets[1])
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

func TestTargetsFromDir(t *testing.T) {
	if TargetsFromDir("/data/vm", 0) != nil {
		t.Fatal("expected nil for n < 1")
	}
	got := TargetsFromDir("/data/vm", 2)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Path != "/data/vm/disk0.img" || got[0].IsBlockDev || got[0].Index != 0 {
		t.Fatalf("disk0: %+v", got[0])
	}
	if got[1].Path != "/data/vm/disk1.img" || got[1].Index != 1 {
		t.Fatalf("disk1: %+v", got[1])
	}
}
