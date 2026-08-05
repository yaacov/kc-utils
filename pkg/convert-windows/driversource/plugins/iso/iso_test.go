package iso

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestExtractVirtioISO(t *testing.T) {
	if _, err := exec.LookPath("bsdtar"); err != nil {
		t.Skip("bsdtar not installed")
	}
	srcDir := t.TempDir()
	driverDir := filepath.Join(srcDir, "NetKVM", "w10", "amd64")
	if err := os.MkdirAll(driverDir, 0o755); err != nil {
		t.Fatal(err)
	}
	inf := filepath.Join(driverDir, "netkvm.inf")
	if err := os.WriteFile(inf, []byte("[version]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "virtio.iso")
	cmd := exec.Command("bsdtar", "-cf", archive, "-C", srcDir, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create archive: %v: %s", err, out)
	}

	dest := t.TempDir()
	if err := extractVirtioISO(archive, dest); err != nil {
		t.Fatal(err)
	}
	got := filepath.Join(dest, "NetKVM", "w10", "amd64", "netkvm.inf")
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("extracted inf missing: %v", err)
	}

	drivers, err := collectDrivers(dest, "x86_64", "Windows 10")
	if err != nil {
		t.Fatal(err)
	}
	if len(drivers) == 0 {
		t.Fatal("expected at least one driver from extracted tree")
	}
}

func TestFindDriversKeepsExtractUntilCleanup(t *testing.T) {
	if _, err := exec.LookPath("bsdtar"); err != nil {
		t.Skip("bsdtar not installed")
	}
	srcDir := t.TempDir()
	driverDir := filepath.Join(srcDir, "viostor", "w10", "amd64")
	if err := os.MkdirAll(driverDir, 0o755); err != nil {
		t.Fatal(err)
	}
	inf := filepath.Join(driverDir, "viostor.inf")
	if err := os.WriteFile(inf, []byte("[version]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sys := filepath.Join(driverDir, "viostor.sys")
	if err := os.WriteFile(sys, []byte("sys"), 0o644); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "virtio.iso")
	cmd := exec.Command("bsdtar", "-cf", archive, "-C", srcDir, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create archive: %v: %s", err, out)
	}

	src := &ISOSource{ISOPath: archive}
	drivers, err := src.FindDrivers("x86_64", "Windows Server 2019")
	if err != nil {
		t.Fatal(err)
	}
	if len(drivers) == 0 {
		t.Fatal("expected drivers from ISO")
	}
	for _, d := range drivers {
		if _, err := os.Stat(d.SrcPath); err != nil {
			t.Fatalf("SrcPath missing before Cleanup: %v", err)
		}
		entries, err := os.ReadDir(d.SrcPath)
		if err != nil {
			t.Fatalf("ReadDir before Cleanup: %v", err)
		}
		if len(entries) == 0 {
			t.Fatal("expected driver files present before Cleanup")
		}
	}

	extractDir := src.extractDir
	if extractDir == "" {
		t.Fatal("extractDir should be set after FindDrivers")
	}
	src.Cleanup()
	if _, err := os.Stat(extractDir); !os.IsNotExist(err) {
		t.Fatalf("extract dir should be removed after Cleanup, err=%v", err)
	}
	if src.extractDir != "" {
		t.Fatalf("extractDir = %q after Cleanup, want empty", src.extractDir)
	}
}
